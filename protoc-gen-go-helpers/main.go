package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

type tmplInput struct {
	Type string
}

const header = `
package %s

import "google.golang.org/protobuf/proto"
`

const helperTmpl = `
func (val *{{.Type}}) Marshal() ([]byte, error) {
    return proto.Marshal(val)
}

func (val *{{.Type}}) Unmarshal(buf []byte) error {
    return proto.Unmarshal(buf, val)
}

// Equal returns whether two {{.Type}} values are equivalent by recursively
// comparing the message's fields.
// For more information see the documentation for
// https://pkg.go.dev/google.golang.org/protobuf/proto#Equal
func (val *{{.Type}}) Equal(other *{{.Type}}) bool {
    if other == nil {
		return val == nil
	}

	other1, ok := other.(*{{.Type}})
	if !ok {
		other2, ok := that.({{.Type}})
		if ok {
			other1 = &other2
		} else {
			return false
		}
	}

    return proto.Equal(val, other)
}`

func main() {
	t := template.Must(template.New("helpers").Parse(helperTmpl))

	// Protoc passes pluginpb.CodeGeneratorRequest in via stdin
	// marshalled with Protobuf
	input, _ := io.ReadAll(os.Stdin)
	var req pluginpb.CodeGeneratorRequest
	proto.Unmarshal(input, &req)

	// Initialise our plugin with default options
	opts := protogen.Options{}
	plugin, err := opts.New(&req)
	if err != nil {
		panic(err)
	}

	// Protoc passes a slice of File structs for us to process
	for _, file := range plugin.Files {

		// Time to generate code...!

		// 1. Initialise a buffer to hold the generated code
		var buf bytes.Buffer

		// 2. Write the package name
		buf.Write([]byte(fmt.Sprintf(header, file.GoPackageName)))

		for _, msg := range file.Proto.MessageType {
			if err := t.Execute(&buf, tmplInput{Type: *msg.Name}); err != nil {
				panic(err)
			}
		}

		file := plugin.NewGeneratedFile(fmt.Sprintf("%s.go-helpers.go", file.GeneratedFilenamePrefix), ".")
		file.Write(buf.Bytes())
	}

	stdout := plugin.Response()
	out, err := proto.Marshal(stdout)
	if err != nil {
		panic(err)
	}

	// Write the response to stdout, to be picked up by protoc
	fmt.Fprintf(os.Stdout, string(out))
}
