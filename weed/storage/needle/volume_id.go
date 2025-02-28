package needle

import (
	"strconv"
)

type VolumeId uint32

func NewVolumeId(vid string) (VolumeId, error) {
	volumeId, err := strconv.ParseUint(vid, 10, 64)

	// now := time.Now()
	// no 15:04:05
	// nowStr := now.Format("20060102") //strings.ReplaceAll(strings.ReplaceAll(now.Format("20060102"), ":", ""), " ", "")
	// nowNumber, _ := strconv.ParseUint(nowStr, 10, 64)
	// fmt.Sprintf("%s%d", strings.ReplaceAll(now.Format("YYYY-MM-DD-HH-MM"), "-", ""),volumeId)
	// nowInNumber := nowNumber*100000 + volumeId
	// log.Println("QUYNGUYEN: NewVolumeId1", nowInNumber, vid, volumeId)

	return VolumeId(volumeId), err
}
func (vid VolumeId) String() string {
	// log.Println("QUYNGUYEN: NewVolumeId string", vid)

	return strconv.FormatUint(uint64(vid), 10)
}
func (vid VolumeId) Next() VolumeId {

	return VolumeId(uint32(vid) + 1)
}
