import math
import json
from fractions import Fraction

import bpyutil
import util


DIR = 'animations'


def deps(context, action, dset):
    dset.add_product((context.path_for_datablock(action), 'Action', action.name))


def cook(context, action):
    fps_num = Fraction(context.bpy_context.scene.render.fps)
    fps_denom = Fraction(context.bpy_context.scene.render.fps_base)
    fps = (fps_num / fps_denom)

    frame_begin = math.floor(action.frame_range[0])
    frame_end = math.ceil(action.frame_range[1])
    duration = frame_end + 1 - frame_begin

    channels = []

    # TODO: also account for scene.render.fps_base (ass)

    # TODO: iiuc layers override "lower levels" so we should just collapse all
    # layers together and output the strips.
    for layer in action.layers:
        for strip in layer.strips:
            for channelbag in strip.channelbags:
                for fcurve in channelbag.fcurves:
                    channels.append({
                        'Name': f'{fcurve.data_path}[{fcurve.array_index}]',
                        'Data': [fcurve.evaluate(i) for i in range(frame_begin, frame_end + 1)],
                    })

    animation = {
        'FrameRate': str(fps.limit_denominator(1001)),
        'AddressMode': 'CLAMP',
        'Duration': duration,
        'Channels': channels,
    }

    animation = bpyutil.fixupdict(animation)

    with context.create(context.path_for_datablock(action)) as f:
        json.dump(animation, util.UTF8Writer(f), default=bpyutil.asdasd, indent='\t')
