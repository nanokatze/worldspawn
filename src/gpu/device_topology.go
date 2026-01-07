package gpu

import (
	"math/bits"
	"slices"

	"worldspawn/gpu/vk"
)

const maxQueueFamilies = 32

// TODO: rename to deviceTopology
type queues struct {
	props  [maxQueueFamilies]vk.QueueFamilyProperties2 // TODO: remove
	flags  [maxQueueFamilies]vk.QueueFlags
	counts [maxQueueFamilies]int
	probe  []uint32 // TODO: should be int
}

func defaultQueues() *queues {
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

	return &queues{
		props:  queueFamilyProps,
		flags:  flags,
		counts: counts,
		probe:  families,
	}
}

// TODO: make these methods be methods on the _Device struct instead and give them better names

func (families *queues) Mask(wantQueueFlags vk.QueueFlags) uint32 {
	var mask uint32
	for i, props := range families.props {
		if props.QueueCount == 0 {
			continue
		}
		if props.QueueFlags&wantQueueFlags == wantQueueFlags {
			mask |= 1 << i
		}
	}
	return mask
}

func (queueFamilies *queues) MinimumCapable(queueFlags vk.QueueFlags) int {
	return 32 - bits.LeadingZeros32(queueFamilies.Mask(queueFlags)) - 1
}

func (families *queues) All() []uint32 {
	return families.probe
}
