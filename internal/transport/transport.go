package transport

type Client interface {
	Call(
		ctx context.Context,
		method string,
		request any,
		response any,
	) error
}
