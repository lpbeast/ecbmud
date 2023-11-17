package combat

import (
	"github.com/lpbeast/ecbmud/mobs"
	"github.com/lpbeast/ecbmud/rooms"
)

func MakeDead(m *mobs.Mob) {
	mZone := rooms.GlobalZoneList[m.Zone]
	mLoc := rooms.GlobalZoneList[m.Zone].Rooms[m.Loc]
	for k, v := range mLoc.Mobs {
		if v == m {
			if k == len(mLoc.Mobs)-1 {
				mLoc.Mobs = mLoc.Mobs[:k]
			} else {
				mLoc.Mobs = append(mLoc.Mobs[:k], mLoc.Mobs[k+1:]...)
			}
		}
	}
	delete(mZone.ActiveMobs, m.ID)
	mZone.DeadMobs[m.ID] = m
}
