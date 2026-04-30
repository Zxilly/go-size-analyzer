import math

import plotille


GRAPH_WIDTH = 80
GRAPH_HEIGHT = 20
X_TICK_SEGMENTS = GRAPH_WIDTH // 10


def _axis_max(values: list[float], segments: int) -> int:
    if not values:
        return segments

    maximum = max(values)
    if maximum <= 0:
        return segments

    return max(segments, math.ceil(maximum / segments) * segments)


def _axis_bounds(values: list[float], segments: int, include_zero: bool = False) -> tuple[int, int]:
    if not values:
        return 0, segments

    minimum = 0 if include_zero else min(values)
    maximum = max(values)
    lower = math.floor(minimum / segments) * segments
    upper = math.ceil(maximum / segments) * segments

    if lower == upper:
        upper = lower + segments

    return lower, upper


def render_usage_graph(
        title: str,
        cpu_percentages: list[float],
        memory_usage_mb: list[float],
        timestamps: list[float],
) -> str:
    x_max = _axis_max(timestamps, X_TICK_SEGMENTS)
    cpu_y_min, cpu_y_max = _axis_bounds(cpu_percentages, GRAPH_HEIGHT, include_zero=True)
    memory_y_min, memory_y_max = _axis_bounds(memory_usage_mb, GRAPH_HEIGHT)

    cpu_graph = plotille.plot(
        timestamps,
        cpu_percentages,
        width=GRAPH_WIDTH,
        height=GRAPH_HEIGHT,
        X_label="Time (seconds)",
        Y_label="CPU %",
        linesep="\n",
        x_min=0,
        x_max=x_max,
        y_min=cpu_y_min,
        y_max=cpu_y_max,
    )
    memory_graph = plotille.plot(
        timestamps,
        memory_usage_mb,
        width=GRAPH_WIDTH,
        height=GRAPH_HEIGHT,
        X_label="Time (seconds)",
        Y_label="Memory (MB)",
        linesep="\n",
        x_min=0,
        x_max=x_max,
        y_min=memory_y_min,
        y_max=memory_y_max,
    )

    return f"{title} CPU %\n{cpu_graph}\n\n{title} Memory (MB)\n{memory_graph}\n"
