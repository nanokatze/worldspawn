package gpu

var resourceDescAlloc = newSlotAlloc(1e6) // TODO: allocate at runtime
var resourceDescAllocHint int64

var samplerSlots = newSlotAlloc(2e3) // TODO: allocate at runtime
var samplerAllocHint int64
