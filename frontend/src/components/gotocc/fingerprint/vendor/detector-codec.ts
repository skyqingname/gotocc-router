/*!
MIT License

Copyright (c) 2026 xqy2006

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
import type { SharedDetector } from './shared-detector'

/**
 * Reads `data/shared_detector.bin`, the one detector file shared by the CLI, the website and the evaluation scripts.
 * `encode_detector` in `offline/export.py` writes it. Layout: u32 header length (little-endian), UTF-8 JSON header,
 * zero padding to 4 bytes, then float32 values. In the header every array of numbers is `{"$f32":offset,"n":length}`;
 * scalars keep full precision. The float bytes are grouped by position (all first bytes, then all second bytes, ...),
 * which zstd compresses much better than interleaved floats; each float is little-endian.
 */
export function decodeDetector(data: Uint8Array): SharedDetector {
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength)
  const headerLength = view.getUint32(0, true), start = Math.ceil((4 + headerLength) / 4) * 4
  const count = (data.length - start) / 4
  const raw = new Uint8Array(count * 4)
  for (let i = 0; i < count; i++) for (let b = 0; b < 4; b++) raw[i * 4 + b] = data[start + b * count + i]
  const little = new DataView(raw.buffer), floats = new Float32Array(count)
  for (let i = 0; i < count; i++) floats[i] = little.getFloat32(i * 4, true)
  return JSON.parse(new TextDecoder().decode(data.subarray(4, 4 + headerLength)), (_, value) =>
    value && typeof value === 'object' && '$f32' in value ? Array.from(floats.subarray(value.$f32, value.$f32 + value.n)) : value)
}
