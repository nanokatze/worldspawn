package gpu

import (
	"math/bits"
	"slices"

	"worldspawn/gpu/vk"
)

const maxQueueFamilies = 32

// TODO: properly newtype this?
type queueFamilyMask = uint32

type deviceTopology struct {
	props  [maxQueueFamilies]vk.QueueFamilyProperties2 // TODO: kill
	flags  [maxQueueFamilies]vk.QueueFlags
	counts [maxQueueFamilies]int
	// TODO: I think probe can be different depending on use case (e.g. for
	// copies crossing pcie we might want to try out the transfer-only queue
	// first, while for other copies we would prefer compute I think.)
	probe []uint32 // TODO: idk if this really belongs here
}

func defaultQueues() *deviceTopology {
	var queueFamilyProps [maxQueueFamilies]vk.QueueFamilyProperties2
	for i := range queueFamilyProps {
		queueFamilyProps[i].SType = vk.STRUCTURE_TYPE_QUEUE_FAMILY_PROPERTIES_2
	}
	n := uint32(len(queueFamilyProps))
	vkFns.GetPhysicalDeviceQueueFamilyProperties2(physicalDevice, &n, &queueFamilyProps[0])

	var flags [maxQueueFamilies]vk.QueueFlags
	for i, props := range queueFamilyProps {
		flags[i] = props.QueueFlags
	}

	var counts [maxQueueFamilies]int
	for i, props := range queueFamilyProps {
		counts[i] = int(props.QueueCount)
	}

	// TODO: order everything
	var families []uint32
	var visited [32]bool
	for _, wantQueueFlags := range []vk.QueueFlags{
		0b111,
		0b110,
		0b100,
	} {
		for family, props := range queueFamilyProps {
			if visited[family] {
				continue
			}
			if props.QueueFlags&wantQueueFlags != wantQueueFlags {
				continue
			}
			families = append(families, uint32(family))
			visited[family] = true
		}
	}
	slices.Reverse(families)

	return &deviceTopology{
		props:  queueFamilyProps,
		flags:  flags,
		counts: counts,
		probe:  families,
	}
}

// TODO: rename
func (topology *deviceTopology) QueueFamilies(flags vk.QueueFlags) queueFamilyMask {
	var mask queueFamilyMask
	for i := range maxQueueFamilies {
		if topology.flags[i]&flags == flags && topology.counts[i] > 0 {
			mask |= 1 << i
		}
	}
	return mask
}

// TODO: kill this
func (topology *deviceTopology) MinimumCapable(queueFlags vk.QueueFlags) int {
	return 32 - bits.LeadingZeros32(topology.QueueFamilies(queueFlags)) - 1
}
