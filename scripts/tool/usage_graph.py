import plotille


def render_usage_graph(
        title: str,
        cpu_percentages: list[float],
        memory_usage_mb: list[float],
        timestamps: list[float],
) -> str:
    cpu_graph = plotille.plot(
        timestamps,
        cpu_percentages,
        width=80,
        height=20,
        X_label="Time (seconds)",
        Y_label="CPU %",
        linesep="\n",
    )
    memory_graph = plotille.plot(
        timestamps,
        memory_usage_mb,
        width=80,
        height=20,
        X_label="Time (seconds)",
        Y_label="Memory (MB)",
        linesep="\n",
    )

    return f"{title} CPU %\n{cpu_graph}\n\n{title} Memory (MB)\n{memory_graph}\n"
