package baseDiceServer

import (
	baseDiceAgent "SOMASExtended/BaseDiceGame/BaseDiceAgent"
	common "SOMASExtended/BaseDiceGame/common"

	baseServer "github.com/MattSScott/basePlatformSOMAS/v2/pkg/server"
//	uuid "github.com/google/uuid"
)

type IBaseDiceServer interface{
	baseServer.IServer[baseDiceAgent.IBaseDiceAgent]
	
}


type BaseDiceServer struct{
	*baseServer.BaseServer[baseDiceAgent.IBaseDiceAgent]
	team common.Team
	

}




