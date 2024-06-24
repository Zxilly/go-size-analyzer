from io import StringIO

import matplotlib
from matplotlib import pyplot as plt

from .svgo import optimize_svg

matplotlib.use('svg')


def draw_usage(
        title: str,
        cpu_percentages: list[float],
        memory_usage_mb: list[float],
        timestamps: list[float]
) -> str:
    buf = StringIO()

    fig, ax1 = plt.subplots(figsize=(14, 5))

    color = 'tab:blue'
    ax1.set_xlabel('Time (seconds)')
    ax1.set_ylabel('CPU %', color=color)
    ax1.plot(timestamps, cpu_percentages, color=color)
    ax1.tick_params(axis='y', labelcolor=color)

    ax1.set_xticks(range(0, int(max(timestamps)) + 1, 1))
    ax1.set_xlim(0, int(max(timestamps)))

    # Add grid to ax1
    ax1.grid(True, which='both', linestyle='--', linewidth=0.5)

    ax2 = ax1.twinx()

    color = 'tab:purple'
    ax2.set_ylabel('Memory (MB)', color=color)
    ax2.plot(timestamps, memory_usage_mb, color=color)
    ax2.tick_params(axis='y', labelcolor=color)

    ax2.set_xticks(ax1.get_xticks())
    ax2.set_xlim(ax1.get_xlim())

    plt.title(f'{title} usage')

    plt.savefig(buf, format='svg')
    buf.seek(0)

    plt.clf()
    plt.close()

    svg = buf.getvalue()
    return optimize_svg(svg)
