package virtualmachineserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"log"
	"net"

	"github.com/achanno/sreapi/certs"
	pb "github.com/achanno/sreapi/protobuf"

	// Needed
	"flag"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	netcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":5555"
	apiv = "v1"

	dbhost = "127.0.0.1:3306"
	dbuser = "sreapi"
	dbpass = "tmp123"
)

// Server t
type Server struct{}

type Virtualmachine struct {
	Hostname string
	Project  string
	Role     string
}

type VirtualmachineInterface interface {
	VirtualmachineFromProto() Virtualmachine
	VirtualmachineFromSQL() Virtualmachine
	VirtualmachineToProto(x Virtualmachine) pb.Virtualmachine
}

func VirtualmachineFromProto() Virtualmachine {

}

var (
	vmEndpoint   = flag.String("vm_endpoint", "localhost:5555", "vm service endpoint")
	db           *sql.DB
	demoKeyPair  *tls.Certificate
	demoCertPool *x509.CertPool
)

func initDBConnection() {
	var err error
	db, err = sql.Open("mysql", dbuser+":"+dbpass+"@tcp("+dbhost+")/sreapi")
	if err != nil {
		log.Fatalf("Error opening db %v", dbhost)
	}
}

// List vms
func (s *Server) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	var err error
	var rows *sql.Rows
	log.Println("List called with project: " + in.Project + " role: " + in.Role)

	if in.Project == "" && in.Role != "" {
		rows, err = db.Query("SELECT * FROM vm WHERE Role like ?", in.Role)
	} else if in.Project != "" && in.Role == "" {
		rows, err = db.Query("SELECT * FROM vm WHERE Project like ?", in.Project)
	} else if in.Project != "" && in.Role != "" {
		rows, err = db.Query("SELECT * FROM vm WHERE Project like ? AND Role like ?", in.Project, in.Role)
	}

	if err != nil {
		log.Println("Error selecting from db: ", err)
	}

	vms := make([]*pb.Virtualmachine, 0)

	for rows.Next() {
		newvm := new(pb.Virtualmachine)
		err = rows.Scan(&newvm.Hostname, &newvm.Project, &newvm.Role)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
		}
		vms = append(vms, newvm)
	}
	rows.Close()

	log.Printf("Addr of vms: %p len: %d data: %v", vms, len(vms), &vms)
	return &pb.ListResponse{XApi: apiv, Vms: vms}, nil
}

// Get vm
func (s *Server) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	vm := new(pb.Virtualmachine)
	log.Println("Get request for: " + in.Hostname)
	rows, err := db.Query("SELECT Hostname, Project, Role  FROM vm WHERE Hostname LIKE ?", in.Hostname)
	defer rows.Close()
	if err != nil {
		log.Printf("Error selcting from db: %v", err)
	}
	rows.Next()
	err = rows.Scan(&vm.Hostname, &vm.Project, &vm.Role)
	if err != nil {
		log.Println("Error scanning row: ", err)
	}
	log.Printf("Found VM: hostname: %s project: %s role: %s", (*vm).Hostname, (*vm).Project, (*vm).Role)
	rows.Close()
	return &pb.GetResponse{XApi: apiv, Vm: vm}, nil
}

// Create vm
func (s *Server) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateResponse, error) {
	log.Println("Creating new vm... hostname: " + in.Hostname + " project: " + in.Project + " role: " + in.Role)

	query := `INSERT INTO vm VALUES (?,?,?)`
	rows, err := db.Query(query, in.Hostname, in.Project, in.Role)

	if err != nil {
		return &pb.CreateResponse{XApi: apiv, Success: false}, err
	}
	rows.Close()
	return &pb.CreateResponse{XApi: apiv, Success: true}, nil
}

// Update vm
func (s *Server) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	log.Println("Updating vm... hostname: " + in.Hostname + " project: " + in.Project + " role: " + in.Role)

	query := "UPDATE vm SET Hostname=?, Project=?, Role=? WHERE Hostname like ?"
	rows, err := db.Query(query, in.Newhostname, in.Project, in.Role, in.Hostname)
	defer rows.Close()
	if err != nil {
		log.Println("Failed updating row: ", err)
		return &pb.UpdateResponse{XApi: apiv, Success: false}, err
	}
	return &pb.UpdateResponse{XApi: apiv, Success: true}, nil
}

// Delete vm
func (s *Server) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	log.Println("Deleting vm: " + in.Hostname)
	query := "DELETE FROM vm WHERE Hostname like ?"
	rows, err := db.Query(query, in.Hostname)
	defer rows.Close()
	if err != nil {
		log.Println("Failed to delete row: ", err)
		return &pb.DeleteResponse{XApi: apiv, Success: true}, err
	}
	return &pb.DeleteResponse{XApi: apiv, Success: true}, nil
}

func grpcHandler(grpcServer *grpc.Server, otherHandler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

// Serve starts grpc server
func Serve(port string) error {
	initDBConnection()
	defer db.Close()

	pair, err := tls.X509KeyPair([]byte(certs.Cert), []byte(certs.Key))

	if err != nil {
		log.Fatalf("Error setting up TLS: %v", err)
	}

	demoKeyPair = &pair
	demoCertPool = x509.NewCertPool()
	ok := demoCertPool.AppendCertsFromPEM([]byte(certs.Cert))
	if !ok {
		log.Fatalf("Error appending certs")
	}

	opts := []grpc.ServerOption{grpc.Creds(credentials.NewClientTLSFromCert(demoCertPool, "localhost:5555"))}
	dcreds := credentials.NewTLS(&tls.Config{
		ServerName: "localhost:5555",
		RootCAs:    demoCertPool,
	})
	dopts := []grpc.DialOption{grpc.WithTransportCredentials(dcreds)}

	netctx := netcontext.Background()
	netctx, cancel := netcontext.WithCancel(netctx)
	defer cancel()

	mux := runtime.NewServeMux()
	err2 := pb.RegisterVirtualmachinesHandlerFromEndpoint(netctx, mux, *vmEndpoint, dopts)
	if err2 != nil {
		log.Fatalf("Error registering endpoint handler: %v", err)
	}

	s := grpc.NewServer(opts...)
	pb.RegisterVirtualmachinesServer(s, &Server{})
	reflection.Register(s)

	srv := &http.Server{
		Addr:    port,
		Handler: grpcHandler(s, mux),
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{*demoKeyPair},
			NextProtos:         []string{"h2"},
			InsecureSkipVerify: true,
		},
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	return srv.Serve(tls.NewListener(lis, srv.TLSConfig))
}
