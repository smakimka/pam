package cli

var CLI struct {
	Reg  RegCmd  `cmd:"" help:"Registration"`
	Auth AuthCmd `cmd:"" help:"Authorization"`
	Rem  RemCmd  `cmd:"" help:"Remember data"`
	Get  GetCmd  `cmd:"" help:"Get data previously remembered"`
	List ListCmd `cmd:"" help:"List all data"`
}
