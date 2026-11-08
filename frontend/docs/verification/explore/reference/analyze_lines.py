"""Measure the reference footer's contour field (frames extracted by ffmpeg)."""
from pathlib import Path
import numpy as np
from PIL import Image, ImageFilter

HERE = Path(__file__).parent
FRAMES = sorted(
    (p for p in (HERE / "frames").glob("t*.png")),
    key=lambda p: float(p.stem[1:]),
)


def gray(path):
    return np.asarray(Image.open(path).convert("L"), dtype=np.float32) / 255.0


def highpass(a, sigma=5):
    blurred = Image.fromarray((a * 255).clip(0, 255).astype(np.uint8)).filter(
        ImageFilter.GaussianBlur(sigma)
    )
    return a - np.asarray(blurred, dtype=np.float32) / 255.0


def dominant_angle(region):
    hp = highpass(region)
    gy, gx = np.gradient(hp)
    mag = np.hypot(gx, gy)
    keep = mag > np.percentile(mag, 90)
    gx, gy = gx[keep], gy[keep]
    jxx, jyy, jxy = (gx * gx).sum(), (gy * gy).sum(), (gx * gy).sum()
    gradient_angle = 0.5 * np.arctan2(2 * jxy, jxx - jyy)
    line_angle = gradient_angle + np.pi / 2
    return np.degrees((line_angle + np.pi / 2) % np.pi - np.pi / 2)


def column_peaks(frame, x, y0=250, y1=560):
    hp = highpass(frame)[y0:y1, x]
    peaks = []
    for i in range(2, len(hp) - 2):
        if hp[i] > hp[i - 1] and hp[i] >= hp[i + 1] and hp[i] > 0.008:
            peaks.append((i + y0, hp[i]))
    # merge peaks closer than 5px, keep the stronger
    merged = []
    for p in peaks:
        if merged and p[0] - merged[-1][0] < 5:
            if p[1] > merged[-1][1]:
                merged[-1] = p
        else:
            merged.append(p)
    return merged


def phase_shift(a, b, max_shift=12):
    """Best vertical (y) and horizontal (x) shift between two crops via FFT."""
    a = highpass(a)
    b = highpass(b)
    fa, fb = np.fft.rfft2(a), np.fft.rfft2(b)
    corr = np.fft.irfft2(fa * np.conj(fb), s=a.shape)
    corr = np.fft.fftshift(corr)
    cy, cx = np.array(corr.shape) // 2
    window = corr[cy - max_shift : cy + max_shift + 1, cx - max_shift : cx + max_shift + 1]
    iy, ix = np.unravel_index(np.argmax(window), window.shape)
    dy = iy - max_shift
    dx = ix - max_shift
    return dx, dy, float(window.max() / (np.abs(corr).max() + 1e-9))


print("frames:", len(FRAMES))
for p in FRAMES:
    f = float(p.stem[1:])
    if f not in (0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11):
        continue
    a = gray(p)
    region = a[250:560, 40:860]
    ang = dominant_angle(region)
    peaks = column_peaks(a, 450)
    gaps = np.diff([p[0] for p in peaks]) if len(peaks) > 2 else []
    print(
        f"t={f:>4}s angle={ang:6.1f}deg peaks@x450={len(peaks):2d} "
        f"median_gap={np.median(gaps) if len(gaps) else float('nan'):.1f}px "
        f"gaps={list(np.round(gaps, 1))}"
    )

# Motion between consecutive 0.5s frames, central clean crop.
print("\nmotion between 0.5s frames (dx, dy, confidence):")
prev = None
for p in FRAMES:
    f = float(p.stem[1:])
    a = gray(p)[300:540, 150:750]
    if prev is not None:
        dx, dy, conf = phase_shift(prev[1], a)
        print(f"t={prev[0]:>4}->{f:<4} dx={dx:+3d} dy={dy:+3d} conf={conf:.2f}")
    prev = (f, a)

# Edge fade: mean line contrast in bands.
print("\ncontrast by band (t=0):")
a = gray(FRAMES[0])
hp = highpass(a)
for name, sl in {
    "left x0-80": (slice(250, 560), slice(0, 80)),
    "mid x400-480": (slice(250, 560), slice(400, 480)),
    "right x820-899": (slice(250, 560), slice(820, 899)),
    "top y0-120": (slice(0, 120), slice(40, 860)),
    "bottom y650-771": (slice(650, 771), slice(40, 860)),
}.items():
    band = a[sl]
    h = hp[sl]
    line_pixels = h[h > 0.006]
    print(
        f"{name:16s} bg={np.percentile(band, 50):.3f} "
        f"p99={np.percentile(band, 99):.3f} hp_p99={np.percentile(h, 99):.4f} "
        f"pixels>0.006={line_pixels.size}"
    )
