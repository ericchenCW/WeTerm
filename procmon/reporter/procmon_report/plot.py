"""Plotly-based multi-process trend charts.

Each chart shows the Top-N services on a single host as separate lines on
the same axes — letting the reader compare per-process usage in context
("which process owns the peak") rather than reading isolated sparklines.

The plotly.js runtime is heavy (~3MB minified). We embed it once at the
top of the report and let every subsequent chart share that runtime —
see ``plotly_runtime_script()`` and the ``include_plotlyjs=False`` flag.
"""
from __future__ import annotations

import colorsys

import pandas as pd
import plotly.graph_objects as go
import plotly.io as pio
from plotly.offline import get_plotlyjs

# Distinct, color-blind friendly palette. ECharts-inspired but accessible.
_PALETTE = [
    "#5470c6", "#91cc75", "#fac858", "#ee6666", "#73c0de",
    "#3ba272", "#fc8452", "#9a60b4", "#ea7ccc", "#5e8a8a",
]


def _hex_to_rgba(hex_color: str, alpha: float) -> str:
    """Convert ``#rrggbb`` to ``rgba(r,g,b,alpha)`` — Plotly accepts the
    string form directly for ``fillcolor``."""
    h = hex_color.lstrip("#")
    r, g, b = int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16)
    return f"rgba({r},{g},{b},{alpha})"


def _darken(hex_color: str, lightness_factor: float = 0.7) -> str:
    """Return a darker variant of ``hex_color`` by scaling its HLS lightness.

    Operates in HLS space (not raw RGB) so hue + saturation stay intact —
    a darker yellow stays yellow rather than turning brown.
    """
    h = hex_color.lstrip("#")
    r, g, b = int(h[0:2], 16) / 255, int(h[2:4], 16) / 255, int(h[4:6], 16) / 255
    hue, lit, sat = colorsys.rgb_to_hls(r, g, b)
    lit = max(0.0, min(1.0, lit * lightness_factor))
    r, g, b = colorsys.hls_to_rgb(hue, lit, sat)
    return f"#{int(r * 255):02x}{int(g * 255):02x}{int(b * 255):02x}"

# Vercel-Analytics-inspired layout: minimal chrome, generous whitespace,
# only a thin horizontal grid, zinc gray axes, no plot border. The visual
# weight stays on the data (lines + gradient fills), not on the frame.
_LAYOUT_DEFAULTS = dict(
    height=320,
    margin=dict(l=52, r=20, t=16, b=36),
    plot_bgcolor="#ffffff",
    paper_bgcolor="#ffffff",
    hovermode="x unified",
    font=dict(
        family='-apple-system, BlinkMacSystemFont, "Inter", "Segoe UI", Helvetica, Arial, sans-serif',
        size=10,
        color="#52525b",  # zinc-600
    ),
    legend=dict(
        orientation="v",
        x=1.0, y=1.0,
        xanchor="left", yanchor="top",
        bgcolor="rgba(255,255,255,0)",
        bordercolor="rgba(0,0,0,0)",
        borderwidth=0,
        font=dict(size=10, color="#71717a"),  # zinc-500
    ),
    xaxis=dict(
        showgrid=False,
        showline=False,    # no axis line
        zeroline=False,
        ticks="",          # no tick marks
        tickfont=dict(size=10, color="#a1a1aa"),  # zinc-400
    ),
    yaxis=dict(
        showgrid=True,
        gridcolor="#f4f4f5",  # zinc-100, barely visible
        gridwidth=1,
        zeroline=False,
        showline=False,
        ticks="",
        tickfont=dict(size=10, color="#a1a1aa"),
    ),
    hoverlabel=dict(
        bgcolor="rgba(24,24,27,0.92)",  # near-black, zinc-900
        bordercolor="rgba(0,0,0,0)",
        font=dict(color="#fafafa", size=11),
    ),
)


def plotly_runtime_script() -> str:
    """Return the full plotly.js library wrapped in a <script> tag.

    Embed this once at the top of the HTML report; the per-chart divs
    below will pick it up without needing to re-bundle 3MB each time.
    """
    js = get_plotlyjs()  # the entire minified plotly.js bundle (~3MB)
    return f"<script>{js}</script>"


def multi_line_chart_html(
    host_df: pd.DataFrame,
    top_keys: list[str],
    value_col: str,
    *,
    value_transform=None,
    y_title: str,
    hover_unit: str,
    hover_fmt: str = ".1f",
) -> str:
    """Render a multi-line plotly chart and return its HTML fragment.

    ``host_df``         filtered to a single host
    ``top_keys``        ordered list of cmdline_keys to plot (top N from a
                        ranking table — first key gets first palette color)
    ``value_col``       column in host_df to plot on Y
    ``value_transform`` optional callable applied per-cell (e.g. KB→MB)
    ``y_title``         human label for the Y axis
    ``hover_unit``      suffix shown after the hover value (e.g. "MB", "%")
    ``hover_fmt``       numeric format for hover values (default 1 decimal)

    Returns an HTML <div>...</div> with embedded data. The plotly runtime
    is NOT embedded — caller must inject ``plotly_runtime_script()`` once.
    """
    fig = go.Figure()
    for i, key in enumerate(top_keys):
        g = host_df[host_df["cmdline_key"] == key].sort_values("ts")
        if g.empty:
            continue
        y = g[value_col]
        if value_transform is not None:
            y = y.apply(value_transform)
        color = _PALETTE[i % len(_PALETTE)]
        fig.add_trace(go.Scatter(
            x=pd.to_datetime(g["ts"], unit="s"),
            y=y,
            mode="lines",
            name=key,
            # Vercel-style: ultra-thin darker line + vertical gradient fill
            # that fades toward the baseline. Each service is independent
            # (no stacking) so a service's trend reads truthfully even
            # when overlapping with others.
            fill="tozeroy",
            line=dict(width=1.0, color=_darken(color, 0.55), shape="linear"),
            fillgradient=dict(
                type="vertical",
                colorscale=[
                    [0.0, _hex_to_rgba(color, 0.02)],  # near-invisible at baseline
                    [1.0, _hex_to_rgba(color, 0.28)],  # subtle at the line
                ],
            ),
            hovertemplate=(
                f"<b>%{{fullData.name}}</b><br>"
                f"%{{y:{hover_fmt}}} {hover_unit}<extra></extra>"
            ),
        ))

    fig.update_layout(**_LAYOUT_DEFAULTS)
    fig.update_yaxes(title_text=y_title, title_font=dict(size=11))

    return pio.to_html(
        fig,
        include_plotlyjs=False,   # the runtime is embedded once at report top
        full_html=False,
        config={
            "displayModeBar": False,  # keep the report clean — no editor UI
            "responsive": True,
        },
    )
