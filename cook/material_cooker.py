import bpy

import material_compiler


# TODO:


# TODO: rename
class Builder:


    def __init__(self, node_tree):
        self.__inedges = {}
        # TODO: there might be a more convenient way to write this
        for l in node_tree.links:
            self.__inedges[l.to_socket] = [l.from_socket]

        self.__instrs = {}

        # TODO: we'll want to collect images with a separate pass, so that after
        # we do various transformations we can avoid including images that
        # aren't used.
        self.__images = []


    def handle_input_socket(self, s, default=None):
        assert not s.is_output
        # TODO: links can be empty, in which case we need to pick the value
        # that's set directly in the socket
        if len(s.links) == 0:
            if default:
                return default()
            return material_compiler.Instr('Constant', [], aux=s.default_value)
        return self.handle_output_socket(self.__inedges[s][0])


    def handle_output_socket(self, s):
        assert s.is_output
        i = self.__instrs.get(s)
        if i is None:
            i = self.__build_instr_for_output_socket(s)
            self.__instrs[s] = i
        return i


    # TODO: rename
    def __build_instr_for_output_socket(self, s):
        n = s.node

        if n.mute:
            for l in n.internal_links:
                if l.to_socket == s:
                    return self.handle_input_socket(l.from_socket)
            assert False, 'unreachable'

        match n:
            case bpy.types.ShaderNodeTexImage():
                assert s.name == 'Color'
                self.__images.append(n.image)
                default_uv = lambda: material_compiler.Instr('Attrib', [])
                return material_compiler.Instr('Image', [
                    self.handle_input_socket(n.inputs[0], default_uv)
                ])
            case bpy.types.ShaderNodeAttribute():
                return material_compiler.Instr('Attrib', [])
            case bpy.types.ShaderNodeVectorMath():
                assert s.name == 'Vector'
                return material_compiler.Instr('Add', [
                    self.handle_input_socket(n.inputs[0]),
                    self.handle_input_socket(n.inputs[1]),
                ])
            case bpy.types.ShaderNodeBsdfPrincipled():
                assert s.name == 'BSDF'
                inputs = {input.name: input for input in n.inputs}
                print(inputs.keys())
                return material_compiler.BSDFPrincipled(
                    BaseColor=self.handle_input_socket(inputs['Base Color']),
                    Metallic=self.handle_input_socket(inputs['Metallic']),
                    Roughness=self.handle_input_socket(inputs['Roughness']),
                    IOR=self.handle_input_socket(inputs['IOR']),
                    Alpha=self.handle_input_socket(inputs['Alpha']),
                )
            case _:
                assert False, 'unhandled node {}'.format(n)


    def get_images(self):
        return self.__images


def print_program(instr, instr_map):
    index = instr_map.get(instr)
    if index is None:
        args = " ".join("%{}".format(print_program(a, instr_map)) for a in instr.args)
        index = len(instr_map) + 1
        print("%{} = {} {}{}".format(index, instr.op, args, instr.aux if instr.aux is not None else ""))
        instr_map[instr] = index
    return index


def cook(context, material):
    # TODO: if this is not a node-based material, print a nice error and bail
    assert material.use_nodes

    node_tree = material.node_tree

    b = Builder(node_tree)

    out = None
    aovs = {}
    for n in node_tree.nodes:
        match n:
            case bpy.types.ShaderNodeOutputMaterial():
                input_socket_map = {input.name: input for input in n.inputs}
                surface = input_socket_map['Surface']
                volume = input_socket_map['Volume']
                if n.is_active_output:
                    out = b.handle_input_socket(surface)
            case bpy.types.ShaderNodeOutputAOV():
                # TODO: we'll want to have a predefined table of AOVs in the
                # exporter, similar to View Layer/Shader AOVs in blender
                aovs[n.aov_name] = b.handle_input_socket(n.inputs[0])

    print(b.get_images())

    instr_map = {}
    print('Material Output:', print_program(out, instr_map))
    for aov_name, aov in aovs.items():
        print(aov_name + ':', print_program(aov, instr_map))
