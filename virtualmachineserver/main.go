package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	pb "sreapi/protobuf"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":5555"
	apiv = "1"

	dbhost = "127.0.0.1:3306"
	dbuser = "sreapi"
	dbpass = "tmp123"
)

// Server t
type Server struct{}

var (
	db *sql.DB
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
	return &pb.ListResponse{Api: apiv, Vms: vms}, nil
	/*	var query, param1 string
		if in.Project == "" && in.Role != "" {
			query = "SELECT * FROM vm WHERE Role LIKE ?"
			param1 = in.Role
		} else if in.Project != "" && in.Role == "" {
			query = "SELECT * FROM vm WHERE Project LIKE ?"
			param1 = in.Project
		} else if in.Project != "" && in.Role != "" {
			query = "SELECT * FROM vm WHERE Project LIKE ? AND Role LIKE ?"
			param1 = in.Project
		}

		rows, err := db.Query(query, param1)
		if err != nil {
			log.Printf("Error running query %v", err)
		}
		vms := make([]*pb.Virtualmachine, 0)
		for rows.Next() {
			vm := new(pb.Virtualmachine)
			err = rows.Scan(&vm.Hostname, &vm.Project, &vm.Role)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
			}
			vms = append(vms, vm)
		}
		return &pb.ListResponse{Api: apiv, Vms: vms}, nil*/
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
	return &pb.GetResponse{Api: apiv, Vm: vm}, nil
}

// Create vm
func (s *Server) Create(ctx context.Context, in *pb.CreateRequest) (*pb.CreateResponse, error) {
	log.Println("Creating new vm... hostname: " + in.Hostname + " project: " + in.Project + " role: " + in.Role)

	query := `INSERT INTO vm VALUES (?,?,?)`
	rows, err := db.Query(query, in.Hostname, in.Project, in.Role)

	if err != nil {
		return &pb.CreateResponse{Api: apiv, Success: false}, err
	}
	rows.Close()
	return &pb.CreateResponse{Api: apiv, Success: true}, nil
}

// Update vm
func (s *Server) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	log.Println("Updating vm... hostname: " + in.Hostname + " project: " + in.Project + " role: " + in.Role)

	query := "UPDATE vm SET Hostname=?, Project=?, Role=? WHERE Hostname like ?"
	rows, err := db.Query(query, in.Newhostname, in.Project, in.Role, in.Hostname)
	defer rows.Close()
	if err != nil {
		log.Println("Failed updating row: ", err)
		return &pb.UpdateResponse{Api: apiv, Success: false}, err
	}
	return &pb.UpdateResponse{Api: apiv, Success: true}, nil
}

// Delete vm
func (s *Server) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	log.Println("Deleting vm: " + in.Hostname)
	query := "DELETE FROM vm WHERE Hostname like ?"
	rows, err := db.Query(query, in.Hostname)
	defer rows.Close()
	if err != nil {
		log.Println("Failed to delete row: ", err)
		return &pb.DeleteResponse{Api: apiv, Success: true}, err
	}
	return &pb.DeleteResponse{Api: apiv, Success: true}, nil
}

func main() {
	initDBConnection()
	defer db.Close()
	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterVirtualmachinesServer(s, &Server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
