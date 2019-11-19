/*
created by Jiahao@harmonycloud  2019/11/5
*/

package main

import (
	"fmt"
	"github.com/containers/storage"
	"github.com/docker/go-units"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/proto"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli"
	"golang.org/x/net/context"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	STORAGEROOT = "/etc/containers/storage.conf"
)

var pidListCommand = cli.Command{
	Name:  "pids",
	Usage: "List process of containers",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "pid, p",
			Value: "",
			Usage: "Filter by pid",
		},
		cli.StringFlag{
			Name:  "state",
			Value: "",
			Usage: "Filter by container state",
		},
		cli.StringFlag{
			Name:  "name",
			Value: "",
			Usage: "filter by container name regular expression pattern",
		},
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "Show all containers",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Output format, One of: json|yaml|table",
		},
	},
	Action: func(context *cli.Context) error {
		var err error
		if err = getRuntimeClient(context); err != nil {
			return err
		}
		if err = getImageClient(context); err != nil {
			return err
		}
		opts := pidListOptions{
			all:        context.Bool("all"),
			pid:        context.String("pid"),
			state:      context.String("state"),
			nameRegexp: context.String("name"),
			output:     context.String("output"),
		}

		if err = pidListContainers(runtimeClient, opts); err != nil {
			return fmt.Errorf("listing containers failed: %v", err)
		}

		return nil
	},
}

func pidListContainers(client pb.RuntimeServiceClient, opts pidListOptions) error {
	filter := &pb.ContainerFilter{}
	st := &pb.ContainerStateValue{}
	if !opts.all {
		st.State = pb.ContainerState_CONTAINER_RUNNING
		filter.State = st
	}
	if opts.state != "" {
		st.State = pb.ContainerState_CONTAINER_UNKNOWN
		switch strings.ToLower(opts.state) {
		case "created":
			st.State = pb.ContainerState_CONTAINER_CREATED
			filter.State = st
		case "running":
			st.State = pb.ContainerState_CONTAINER_RUNNING
			filter.State = st
		case "exited":
			st.State = pb.ContainerState_CONTAINER_EXITED
			filter.State = st
		case "unknown":
			st.State = pb.ContainerState_CONTAINER_UNKNOWN
			filter.State = st
		default:
			log.Fatalf("--state should be one of created, running, exited or unknown")
		}
	}

	request := &pb.ListContainersRequest{
		Filter: filter,
	}
	r, err := client.ListContainers(context.Background(), request)
	if err != nil {
		return err
	}

	switch opts.output {
	case "json":
		return outputAsJSON(r)
	case "yaml":
		return outputAsYAML(r)
	case "table", "":
	// continue; output will be generated after the switch block ends.
	default:
		return fmt.Errorf("unsupported output format %q", opts.output)
	}

	display := newTableDisplay(20, 1, 3, ' ', 0)
	display.AddRow([]string{columnContainer, columnCreated, columnState, columnName, columnPID, columnIP, columnMountPoint})

	storageOpts := storage.StoreOptions{}
	storage.ReloadConfigurationFile(STORAGEROOT, &storageOpts)

	root := filepath.Join(storageOpts.GraphRoot, storageOpts.GraphDriverName+"-containers")
	for _, c := range r.Containers {
		if !matchesRegex(opts.nameRegexp, c.Metadata.Name) {
			continue
		}
		createdAt := time.Unix(0, c.CreatedAt)
		ctm := units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
		id := c.Id
		configRoot := filepath.Join(root, id, "userdata", "config.json")
		stateRoot := filepath.Join(root, id, "userdata", "state.json")
		configJson, err := ioutil.ReadFile(configRoot)
		if err != nil {
			return err
		}
		stateJson, err := ioutil.ReadFile(stateRoot)
		if err != nil {
			return err
		}
		mountPoint := gjson.Get(string(configJson), "root.path").String()
		pid := gjson.Get(string(stateJson), "pid").String()
		IP := gjson.Get(string(stateJson), "annotations").Get("io.kubernetes.cri-o.IP").String()

		display.AddRow([]string{getTruncatedID(id, ""), ctm, convertContainerState(c.State), c.Metadata.Name,
			pid, IP, mountPoint})
	}
	_ = display.Flush()
	return nil
}

func outputAsJSON(obj proto.Message) error {
	marshaledJSON, err := protobufObjectToJSON(obj)
	if err != nil {
		return err
	}

	fmt.Println(marshaledJSON)
	return nil
}

func outputAsYAML(obj proto.Message) error {
	marshaledJSON, err := protobufObjectToJSON(obj)
	if err != nil {
		return err
	}
	marshaledYAML, err := yaml.JSONToYAML([]byte(marshaledJSON))
	if err != nil {
		return err
	}

	fmt.Println(string(marshaledYAML))
	return nil
}
