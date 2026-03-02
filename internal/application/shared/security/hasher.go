package security

type Hasher interface {
	Hash(str string) (string, error)
	HashBytes(bytes []byte) (string, error)
	Verify(str, hash string) bool
}
