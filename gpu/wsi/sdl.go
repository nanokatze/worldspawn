package gpuwsi

// TODO: rename this package to sdlwsi or wsi_sdl or whatever

// TODO: worldspawn/gpu should not import this package, it should be usable by
// the end users, so it will need an API makeover as well

// #cgo pkg-config: sdl3
//
// #include <SDL3/SDL.h>
// #include <SDL3/SDL_vulkan.h>
import "C"

import (
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/internal/sdl"
)

func init() {
	// TODO: figure out some way to get these from sdl reliably. I guess we
	// could just poke C.SDL_Init or whatever.
	gpu.WantInstanceExtension("VK_KHR_surface")
	gpu.WantInstanceExtension("VK_KHR_wayland_surface")
	gpu.WantInstanceExtension("VK_KHR_xlib_surface")

	gpu.WantDeviceExtension("VK_KHR_swapchain")
	gpu.WantDeviceExtension("VK_KHR_swapchain_mutable_format")
}

// TODO: hide
/*
func InstanceExtensions() []string {
	if C.SDL_WasInit(C.SDL_INIT_VIDEO) == 0 {
		return nil
	}

	var n C.Uint32
	cexts := C.SDL_Vulkan_GetInstanceExtensions(&n)

	exts := make([]string, n)
	for i, cext := range unsafe.Slice(cexts, n) {
		exts[i] = C.GoString(cext)
	}
	return exts
}
*/

func sdlVulkanCreateSurface(window *sdl.Window, instance vk.Instance, allocator *vk.AllocationCallbacks) (vk.SurfaceKHR, error) {
	var surface vk.SurfaceKHR
	if !C.SDL_Vulkan_CreateSurface(
		(*C.SDL_Window)(window),
		*(*C.VkInstance)(unsafe.Pointer(&instance)),
		nil, // TODO: actually pass allocator
		(*C.VkSurfaceKHR)(unsafe.Pointer(&surface))) {
		// TODO: get + return the error
		panic("SDL_Vulkan_CreateSurface failed")
	}
	return surface, nil
}
