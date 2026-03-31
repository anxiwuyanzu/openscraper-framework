//go:build windows

package dot

func NewSinker(kafkaServers string) ISinker {
	return NewDebugSinker(Conf().Sinker)
}

// TODO: 适用于windows的kafka库
