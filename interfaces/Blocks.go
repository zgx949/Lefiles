package interfaces

type BlockStorage interface {
	ReadBlock(path string) ([]byte, error)
	WriteBlock(path string, block []byte) error
}
