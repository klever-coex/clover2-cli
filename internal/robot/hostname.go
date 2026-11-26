package robot

type HostnameService struct{}

func (s *HostnameService) Get(ctx context.Context) (string, error) {
	return os.Hostname()
}

func (s *HostnameService) Set(ctx context.Context, hostname string) error {
	return nil
}
