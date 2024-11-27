package common
package common
import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/message"
	"github.com/google/uuid"
)

// IExtendedMessage defines the interface for messages in the system
type IExtendedMessage interface {
	message.IMessage[IExtendedAgent]
	GetTeamID() uuid.UUID
}
