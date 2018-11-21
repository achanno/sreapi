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

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
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

var (
	vmEndpoint   = flag.String("vm_endpoint", "localhost:5555", "vm service endpoint")
	db           *gorm.DB
	demoKeyPair  *tls.Certificate
	demoCertPool *x509.CertPool
)

// Sreapii interface used to convert between databse entries and protobuf messages
type Sreapii interface {
	toDB() interface{}
	toProto() interface{}
}

type projectdb struct {
	gorm.Model
	Name string
}

type project pb.Project
type stack pb.Stack
type role pb.Role
type vm pb.Virtualmachine

func (p *projectdb) toDB() interface{} {
	return p
}

func (p *projectdb) toProto() interface{} {
	return &pb.Project{Name: p.Name}
}

func (p *project) toDB() interface{} {
	return &projectdb{Name: p.Name}
}

func (p *project) toProto() interface{} {
	return p
}

type stackdb struct {
	gorm.Model
	Project string
	Name    string
}

func (p *stackdb) toDB() interface{} {
	return p
}
func (p *stackdb) toProto() interface{} {
	return &pb.Stack{name: p.name}
}

func (p *stack) toDB() interface{} {
	project = "test" // Grab from db?
	return &stackdb{Name: p.Name, Project: project}
}

func (p *stack) toProto() interface{} {
	return p
}

type roledb struct {
	gorm.Model
	Project    string
	Stack      string
	Name       string
	ParentRole string
}

func (p *roledb) toDB() interface{} {
	return p
}

func (p *roledb) toProto() interface{} {
	return &pb.Stack{name: p.Name}
}

func (p *role) toDB() interface{} {
	return &roledb{Name: p.name, Stack: "test", Project: "test", ParentRole: "someotherrole"}
}
func (p *role) toProto() interface{} {
	return p
}

type vmdb struct {
	gorm.Model
	Project  string
	Stack    string
	Role     string
	Hostname string
}

func (p *vmdb) toDB() interface{} {
	return p
}
func (p *vmdb) toProto() interface{} {
	return &pb.Virtualmachine{Hostname: p.Hostname}
}

func (p *vm) toDB() interface{} {
	return &vmdb{Hostname: p.Hostname, Project: "test", stack: "test", Role: "Test"}
}

func (p *vm) toProto() interface{} {
	return p
}

func initDBConnection() {
	var err error
	db, err = sql.Open("mysql", dbuser+":"+dbpass+"@tcp("+dbhost+")/sreapi")
	if err != nil {
		log.Fatalf("Error opening db %v", dbhost)
	}
}

// ProjectServer t
type ProjectServer struct{}

// StackServer t
type StackServer struct{}

// RoleServer t
type RoleServer struct{}

// VMServer t
type VMServer struct{}

// List Projects
func (s *ProjectServer) List(ctx context.Context, in *pb.ListProjectRequest) (*pb.ListProjectResponse, error) {
	var results []projectdb
	db.Find(&results)
	var resultsproto []pb.Project
	for x := range results {
		resultsproto = append(resultsproto, Sreapii.toProto(results[x]))
	}
	return &pb.ProjectListResponse{XApi: apiv, Projects: resultsproto}, nil
}

// List Stacks
func (s *StackServer) List(ctx context.Context, in *pb.ListStackRequest) (*pb.ListStackResponse, error) {
	return nil, nil
}

// List Roles
func (s *RoleServer) List(ctx context.Context, in *pb.ListRoleRequest) (*pb.ListRoleResponse, error) {
	return nil, nil
}

// List vms
func (s *VMServer) List(ctx context.Context, in *pb.ListVMRequest) (*pb.ListVMResponse, error) {
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
