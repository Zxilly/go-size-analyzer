from io import BytesIO

import matplotlib
from matplotlib import pyplot as plt

matplotlib.use('agg')


def draw_usage(
        cpu_percentages: list[float],
        memory_usage_mb: list[float],
        timestamps: list[float]
) -> bytes:
    buf = BytesIO()

    fig, ax1 = plt.subplots(figsize=(14, 5))

    color = 'tab:blue'
    ax1.set_xlabel('Time (seconds)')
    ax1.set_ylabel('CPU %', color=color)
    ax1.plot(timestamps, cpu_percentages, color=color)
    ax1.tick_params(axis='y', labelcolor=color)

    ax1.set_xticks(range(0, int(max(timestamps)) + 1, 1))
    ax1.set_xlim(0, int(max(timestamps)))

    ax2 = ax1.twinx()

    color = 'tab:purple'
    ax2.set_ylabel('Memory (MB)', color=color)
    ax2.plot(timestamps, memory_usage_mb, color=color)
    ax2.tick_params(axis='y', labelcolor=color)

    plt.title('CPU and Memory Usage')

    plt.savefig(buf, format='svg')
    buf.seek(0)

    plt.clf()
    plt.close()

    return buf.getvalue()
