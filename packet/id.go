package packet

const (
	HandshakeID        int32 = 0
	StatusID           int32 = 0
	PingID             int32 = 1
	PongID             int32 = 1
	DisconnectID       int32 = 27
	LegacyDisconnectID int32 = 26 // disconnect packet id before 1.20.2
	LoginStartID       int32 = 0
	LoginDisconnectID  int32 = 0
	LoginEncryptionID  int32 = 1
	LoginSuccessID     int32 = 2
	LoginCompressionID int32 = 3
	LoginPluginID      int32 = 4
)
