# **Asset Cooker**

* Consider replacing JSON header with something like protobuf header instead?
  The blob tail would still remain as is.

* We might want a convenience option to check individual objects (hierarchies)
  to be exported as prefabs, as having the entire scene be a prefab is not
  necessarily intuitive

* Explore interleaving user attributes, instead of exporting them as SOA?

* Export materials **from blender**, instead of authoring them separately and
  then importing into blender. The game will be configuring certain nodes at
  runtime by referencing their name.

  * We need something to compute correct motion vectors for animated
    materials, or an option to mask these areas out outright, so that TAA
    does't get confused.

    The users can use an appropriately named AOV to export the correct motion
    vectors, or TAA mask out.

  * We have several requirements for the whole material thing:

    1. We should strive to minimize number of API shaders (shaderlets) we use
        to implement materials. Given materials A and B, if shaderlet for A is
        more general than that of B, we should be using A"s shaderlet to
        implement B (unless it's so much more general that it's likely to
        regress perf.)

    2. We should enable relatively straightforward manual intervention into
        the whole material to shaderlet mapping as well as encoding and layout
        of the parameters. RTX Remix for example packs material parameters into
        two uvec4 for performance reasons, by assuming lots of things like IOR
        being only between 1 and 3.

* Object linking issues: modifiers are per-object, but we export geometry
  *after* modifiers. Possible solutions include:

  * Creating a proxy object with modifiers applied on save, and then linking
    these proxy objects.

  * Requiring that the user links objects that already have no modifiers. This
    may increase friction on export but is an elegant solution otherwise...

# Client and server

* Make the transform hierarchy actually work, this is especially necessary for
  viewmodels

* Clean up the code in both client and server

* Improve allocator in our GPU abstraction

# Rendering

* Prioritize sky (IBL) sampling over all other types of lights

  * We should compute various maps for improving sampling at runtime, rather
    than offline, as we could be generating the skybox at runtime

  * Of other lights, we'll want to implement portals first and foremost

  * Not sure how we should deal with other lights. Ideally we'd ignore all of
    that and only care about emissive polygons and volumes?

* Implement dithering or at least provide a basic setup. This one should be
  pretty easy.

* Implement AgX or more general LUT-based tonemapper? This should be pretty easy
  too I think...

* Rasterize primary rays? Not clear how worth this actually is. This will
  require extra effort to make work with micromaps. Rasterizing micromaps might
  also diminish its performance with respect to RT.

* DDGI for diffuse GI?

* Implement SVGF once we have indirect lighting going.

* Glossy GI TBD but will probably either use denoised RT or SSR or mixture of
  both. Actually, why don't we just avoid glossiness in our content?

* Implement RESTIR and REGIR for better samples

* Build a dependency graph so that we know what things in renderer's scene to
  update when a single component in the game state changes.

* The renderer computes scene by interpolating between two game states. If we
  teleport an entity this will cause it to travel along the long path. We should
  have some machinery in place to introduce discontinuities. One heuristic would
  be comparing how far end position position is from starting position plus
  starting velocity times delta time.

  * When composing transforms we might get shearing, so we'll need to think
    about that as well.

    * Our current solution is to remove Affine3 composition. Instead, we will
      convert to matrices and do composition in matrices.

    * Still, even if we're not going to implement composition in Affine3, we
      probably should still make Affine3 be able to represent shearing just for
      completeness.

  * Maybe interpolate between more states? Source 2 seems to interpolate between 3
    frames now. Not sure if useful.

# Sound

* **Implement sphere sampling**

* Make our sound code not be utter garbage that spams syscalls

# Physics

* Change physics package to be just roughly raw bindings to jolt physics.

  * Get rid of C geometry package and move everything necessary into jolt physics

# Uncategorized

* Consider migrating to CMake? Our only native non-system dependency
  (JoltPhysics) uses CMake so we probably don't benefit from meson much. Figure
  out if CMake has some sort of nice support for Go...

* Migrate off git LFS to just plain git. Perhaps raw content should live in its
  own repo, linked as submodule?
