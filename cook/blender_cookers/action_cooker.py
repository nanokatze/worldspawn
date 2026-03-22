import math
import json


import bpyutil
import util


def deps(context, action, dset):
    dset.add_product((context.path_for_datablock(action), 'Action', action.name))


def cook(context, action):
    frame_begin = math.floor(action.frame_range[0])
    frame_end = math.ceil(action.frame_range[1])

    channels = {}

    # TODO: iiuc layers override "lower levels" so we should just collapse all
    # layers together and output the strips.
    for layer in action.layers:
        for strip in layer.strips:
            for channelbag in strip.channelbags:
                for fcurve in channelbag.fcurves:
                    key = f'{fcurve.data_path}[{fcurve.array_index}]'

                    channels[key] = [fcurve.evaluate(i) for i in range(frame_begin, frame_end+1)]

    animation = {
        'Frames': frame_end+1-frame_begin,
        'Channels': channels,
    }

    animation = bpyutil.fixupdict(animation)

    with open(context.path_for_datablock(action), 'wb') as f:
        json.dump(animation, util.UTF8Writer(f), default=bpyutil.asdasd, indent='\t')
