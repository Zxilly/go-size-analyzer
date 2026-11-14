package main

type Node struct {
	Text string
	Next *Node
}
type Payload struct {
	Text  string
	Bytes [256]byte
}

var (
	Root          = &Node{Text: "static_pointer_payload_8e5fd0d1b7b04b3fa108", Next: &Shared}
	Shared        = Node{Text: "shared_pointer_payload_f10ec8834459460bb9d6"}
	CycleA        = Node{Text: "cycle_a_payload_24a6ba671f5e4bd19ffc", Next: &CycleB}
	CycleB        = Node{Text: "cycle_b_payload_9f43423ae37d47749d48"}
	Box       any = "interface_box_payload_235fad16b29b43f2"
	StructBox any = Payload{Text: "dynamic_interface_payload_e8b0c3"}
	Direct    any = &Shared
)

// This back-edge is dynamic and must not be invented by a static file scan.
func init() { CycleB.Next = &CycleA }

func cross()
func main() {
	// Built-in printing preserves the controlled fixture without importing fmt.
	//nolint:revive // use-fmt-print: changing imports changes the measured binary.
	println(Root.Text, Root.Next.Text, CycleA.Next.Text, Box.(string), StructBox.(Payload).Text, Direct.(*Node).Text)
	cross()
}
