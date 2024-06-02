package main

import "fmt"

type Sim struct {
	Reg [8]uint16
}

func (s *Sim) Execute(instruction *Instruction) error {
	// TODO
	return nil
}

func (s *Sim) PrintRegisters() {
	fmt.Println("Final registers:")
	// TODO
}
