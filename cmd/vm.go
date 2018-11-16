package cmd

import (
	"errors"
	pb "github.com/achanno/sreapi/protobuf"
	vmserver "github.com/achanno/sreapi/virtualmachineserver"
	"github.com/spf13/cobra"
	"log"
)

var (
	project string
	role    string
)

// VMDeleteCommandFunc r
func VMDeleteCommandFunc(cmd *cobra.Command, args []string) {
	r, err := c.Delete(ctx, &pb.DeleteRequest{XApi: apiv, Hostname: args[0]})
	if err != nil {
		log.Fatalf("Could not delete vm: %v", err)
	}
	log.Print("Deleted: ", args[0], " Sucess: ", r.Success)
}

// VMUpdateCommandFunc r
func VMUpdateCommandFunc(cmd *cobra.Command, args []string) {
	r, err := c.Update(ctx, &pb.UpdateRequest{XApi: apiv, Hostname: args[1], Project: args[2], Role: args[3], Oldhostname: args[0]})
	if err != nil {
		log.Fatalf("Could not list vms: %v", err)
	}
	log.Print("Updated: ", args[0], r.Success)
}

// VMCreateCommandFunc r
func VMCreateCommandFunc(cmd *cobra.Command, args []string) {
	r, err := c.Create(ctx, &pb.CreateRequest{XApi: apiv, Hostname: args[0], Project: args[1], Role: args[2]})
	if err != nil {
		log.Fatalf("Could not create vm: %v", err)
	}
	log.Print("Created VM: ", r.Success)
}

// VMGetCommandFunc r
func VMGetCommandFunc(cmd *cobra.Command, args []string) {
	r, err := c.Get(ctx, &pb.GetRequest{XApi: apiv, Hostname: args[0]})
	if err != nil {
		log.Fatalf("Could not list vms: %v", err)
	}
	log.Print("Found stuff: ", r.Vm.Hostname, r.Vm.Project, r.Vm.Role)
}

// VMListCommandFunc r
func VMListCommandFunc(cmd *cobra.Command, args []string) {
	if project != "" {
		log.Print("Found project: ", project)
	}

	if role != "" {
		log.Print("Found role: ", role)
	}
	var r *pb.ListResponse
	var errmsg error
	if project != "" && role != "" {
		r, errmsg = c.List(ctx, &pb.ListRequest{XApi: apiv, Role: role, Project: project})
	} else if project != "" && role == "" {
		r, errmsg = c.List(ctx, &pb.ListRequest{XApi: apiv, Project: project})
	} else if project == "" && role != "" {
		r, errmsg = c.List(ctx, &pb.ListRequest{XApi: apiv, Role: role})
	} else {
		log.Fatalf("Requires --project or --role")
	}

	if errmsg != nil {
		log.Fatalf("Could not list vms: %v", errmsg)
	}

	for x := range r.Vms {
		log.Printf("Hostname: %s Project: %s Role: %s", r.Vms[x].Hostname, r.Vms[x].Project, r.Vms[x].Role)
	}
}

// VMServerCommandFunc r
func VMServerCommandFunc(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		vmserver.Serve(":" + args[0])
	} else {
		vmserver.Serve(":5555")
	}
}

// VMCreateCommand r
func VMCreateCommand() *cobra.Command {
	vmcommand := &cobra.Command{
		Use:   "create <hostname> <project> <role>",
		Short: "Creates new vm",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return errors.New("create requires <hostname> <project> <role>")
			}
			return nil
		},
		Run: VMCreateCommandFunc,
	}
	return vmcommand
}

// VMListCommand r
func VMListCommand() *cobra.Command {
	vmcommand := &cobra.Command{
		Use:   "list <project/role>",
		Short: "Lists vms related to projects roles",
		Run:   VMListCommandFunc,
	}

	vmcommand.Flags().StringVar(&project, "project", "", "project name")
	vmcommand.Flags().StringVar(&role, "role", "", "role name")
	return vmcommand
}

// VMGetCommand r
func VMGetCommand() *cobra.Command {
	vmcommand := &cobra.Command{
		Use:   "get <hostname>",
		Short: "Get vm for hostname",
		Run:   VMGetCommandFunc,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("get requires <hostname>")
			}
			return nil
		},
	}
	return vmcommand
}

// VMUpdateCommand r
func VMUpdateCommand() *cobra.Command {
	vmcommand := &cobra.Command{
		Use:   "update <oldhostname> <hostname> <project> <role>",
		Short: "Update vm",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 4 {
				return errors.New("update requires <oldhostname> <hostname> <project> <role>")
			}
			return nil
		},
		Run: VMUpdateCommandFunc,
	}
	return vmcommand
}

// VMDeleteCommand r
func VMDeleteCommand() *cobra.Command {
	vmcommand := &cobra.Command{
		Use:   "delete <hostname>",
		Short: "Deletes a vm",
		Args:  cobra.ExactArgs(1),
		Run:   VMDeleteCommandFunc,
	}
	return vmcommand
}

// VMServerCommand r
func VMServerCommand() *cobra.Command {
	vmcommand := &cobra.Command{
		Use:   "server <port>",
		Short: "Start server on <port>",
		Args:  cobra.MaximumNArgs(1),
		Run:   VMServerCommandFunc,
	}
	return vmcommand
}

// VMCommand r
func VMCommand() *cobra.Command {
	vmcmd := &cobra.Command{
		Use:   "vm <subcommand>",
		Short: "vm related commands",
	}

	vmcmd.AddCommand(VMCreateCommand())
	vmcmd.AddCommand(VMListCommand())
	vmcmd.AddCommand(VMGetCommand())
	vmcmd.AddCommand(VMUpdateCommand())
	vmcmd.AddCommand(VMDeleteCommand())
	vmcmd.AddCommand(VMServerCommand())
	return vmcmd
}

func init() {
	rootCmd.AddCommand(VMCommand())
}
