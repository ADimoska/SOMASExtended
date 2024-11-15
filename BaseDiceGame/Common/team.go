package common

import "github.com/google/uuid"


type Team struct{
	teamID uuid.UUID
	commonPool int
	agents []uuid.UUID
	strategy int
 
}

 


