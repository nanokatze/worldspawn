package gpu

import (
	"slices"
	"testing"

	"worldspawn/gpu/vk"
)

func TestQueueFamilySelection(t *testing.T) {
	for i, test := range []struct {
		queueFlags      []vk.QueueFlags
		expectedIndices []uint32
	}{
		{[]vk.QueueFlags{0b111}, []uint32{0}},
		{[]vk.QueueFlags{0b111, 0b100}, []uint32{1, 0}},
		{[]vk.QueueFlags{0b111, 0b110}, []uint32{1, 0}},
		{[]vk.QueueFlags{0b111, 0b100, 0b110}, []uint32{1, 2, 0}},
		{[]vk.QueueFlags{0b111, 0b110, 0b100}, []uint32{2, 1, 0}},
	} {
		props := make([]vk.QueueFamilyProperties2, len(test.queueFlags))
		for i, queueFlags := range test.queueFlags {
			props[i] = vk.QueueFamilyProperties2{
				SType: vk.STRUCTURE_TYPE_QUEUE_FAMILY_PROPERTIES_2,
				QueueFamilyProperties: vk.QueueFamilyProperties{
					QueueFlags:                  queueFlags,
					QueueCount:                  1,
					TimestampValidBits:          64,
					MinImageTransferGranularity: vk.Extent3D{Width: 1, Height: 1, Depth: 1},
				},
			}
		}

		got := chooseQueueFamilies2(props)
		want := test.expectedIndices
		if !slices.Equal(got, want) {
			t.Errorf("%d: got = %v, want = %v", i, got, want)
		}
	}
}
