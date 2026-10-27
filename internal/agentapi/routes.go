package agentapi

const APIPrefix = "/api/v1"

const (
	HealthRoute = APIPrefix + "/health"
	InfoRoute   = APIPrefix + "/info"
)

const (
	DockerRoute = APIPrefix + "/docker"

	DockerImagesRoute     = DockerRoute + "/images"
	DockerContainersRoute = DockerRoute + "/containers"

	DockerImageItemsRoute   = DockerRoute + "/items"
	DockerImageRoutePattern = DockerImagesRoute + "/{id}"
	DockerImageInfoPattern  = DockerImageRoutePattern

	DockerContainerItemsRoute   = DockerRoute + "/items"
	DockerContainerRoutePattern = DockerContainersRoute + "/{id}"
	DockerContainerInfoPattern  = DockerContainerRoutePattern
)
