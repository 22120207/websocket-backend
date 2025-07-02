package models

type SSHConfig struct {
	Target   string
	Username string
	Password string
	KeyPath  string
}

func (s *SSHConfig) UsePrivateKey() bool {
	return s.KeyPath != ""
}
