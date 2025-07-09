import bpy


# TODO: when serializing we'd want to have things work in a backwards-compatible
# manner, so we'll need to have a schema table for instrs or something.


# TODO: rename to Value?
class Instruction:


    def __init__(self, op, type, args, aux=None):
        self._op = op
        self._type = type
        self._args = args
        self._aux = aux
        # self._di = None # stick source node here


    # TODO: implement hash


# TODO: we also need a description of aux
INSTRUCTION_SCHEMA = {
    'GetAttribute': [],
    'GetTexture': [],
    'GetSampler': [],
    'SampleFlat': ['Texture', 'Sampler', 'Point'],
    'BSDFPrincipled': ['BaseColor'],
}


# TODO: use enum for types
# TODO: use numeric enum instead of strings for ops


# TODO: all of these should go through CSE facility (and probably rewrite rules too)
# TODO: we probably don't need to provide builders for all instructions, only for certain.
# TODO: attach debug info (node name, etc) to instructions
class Builder:

    def __init__(self):
        self._cse = {}

    # TODO: rename to new or something else?
    def build(self, op, type, args, aux=None):
        # TODO: do CSE hit here
        return Instruction(op, type, args, aux)

    # TODO: move helpers out of this type
    def new_get_sampler(self, interpolation_mode, extension_mode):
        return self.build('GetSampler', 'Sampler', [], aux=(interpolation_mode, extension_mode))

    # TODO: this also needs a "default value"
    def new_get_texture(self, name):
        return self.build('GetTexture', 'SampledImageView', [], aux=name)

    def new_get_attribute(self, type, name):
        return self.build('GetAttribute', type, [], aux=name)

    # TODO: replace texture+sampler with a single param and add a
    # combine_texture_with_sampler instr?
    def new_sample_flat(self, texture, sampler, P):
        return self.build('SampleFlat', 'Vec4', [texture, sampler, P])


def print_instrs(name_map, instr):
    for a in instr._args:
        print_instrs(name_map, a)
    name = name_map.setdefault(instr, len(name_map))

    schema = INSTRUCTION_SCHEMA[instr._op]

    assert len(instr._args) == len(schema)

    args_stringified = ' '.join(f'{schema[i]}=%{name_map[a]}' for (i, a) in enumerate(instr._args))

    print(f'%{name} {instr._type} = {instr._op} {instr._aux} {args_stringified}')


def test(material):
    # TODO: examine nodes and capture the "shader" (part of the material
    # that we want to turn into the API shader.)
    #
    # We'll work from the currently active material output and work our
    # way backwards to build the "shader".

    node_tree = material.node_tree

    # TODO: maybe just get an index, that would be nicer probably
    # TODO: we also need to find and walk backwards from AOVs in addition to this
    active_output = next(node for node in node_tree.nodes if isinstance(node, bpy.types.ShaderNodeOutputMaterial) and node.is_active_output)

    # TODO: probably abort and produce diagnostic when we encounter a
    # None active_output

    b = Builder()

    def get_instruction_for_blender_node(node):
        match node:
            case bpy.types.ShaderNodeTexImage():
                match node.projection:
                    case 'FLAT':
                        # TODO: how can we let the user change texture at runtime?
                        # TODO: proper ktx2 mangling
                        # TODO: export color space (we can only support sRGB and non-color/linear)
                        instruction = b.new_sample_flat(
                            b.new_get_texture(node.image.filepath[:-4] + '.ktx2'),
                            b.new_get_sampler(node.interpolation, node.extension), # TODO: translate these to our enums
                            b.new_get_attribute('Vec2', 'UVMap.default')) # better name for default uv map

                    # TODO: different instructions for different projections

                    case _:
                        assert False, f'unsupported projection {node.projection}'

            case bpy.types.ShaderNodeBsdfPrincipled():
                base_color = walk(node.inputs[0].links[0].from_node)

                # TODO: maybe we should decompose BSDFPrincipled into many
                # things? Probably not...
                instruction = b.build('BSDFPrincipled', 'Shader', [base_color])

            # TODO: same for AOV nodes
            case bpy.types.ShaderNodeOutputMaterial():
                assert False, 'should not get here'

            case _:
                assert False, f'unsupported node of type {type(node)}'

    visited = dict()
    def walk(node):
        if node in visited:
            return visited[node]

        instruction = get_instruction_for_blender_node(node)

        visited[node] = instruction
        return instruction

    assert isinstance(active_output, bpy.types.ShaderNodeOutputMaterial)
    surface = walk(active_output.inputs[0].links[0].from_node)

    # TODO: serialize this into json or smth I guess?
    print_instrs({}, surface)

    # print(f'"{link.from_node.name}" -> "{link.to_node.name}"."{link.to_socket.identifier}"')

    # TODO: respect muting the nodes as well. We'll need to look at
    # internal_links for that
