package daemon

import "embed"

//go:embed all:ui/host
var hostFS embed.FS

//go:embed all:ui/participant
var participantFS embed.FS

const hostUIRoot = "ui/host"
const participantUIRoot = "ui/participant"
