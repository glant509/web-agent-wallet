export function createQRCodeMatrix(value: string): boolean[][] {
  const version = 5;
  const size = version * 4 + 17;
  const dataCodewords = 108;
  const errorCodewords = 26;
  const bytes = Array.from(new TextEncoder().encode(value));
  if (bytes.length > 106) throw new Error("接收地址过长，无法生成二维码。");
  const bits: number[] = [];
  const appendBits = (number: number, length: number) => {
    for (let bit = length - 1; bit >= 0; bit -= 1) bits.push((number >>> bit) & 1);
  };
  appendBits(4, 4);
  appendBits(bytes.length, 8);
  bytes.forEach((byte) => appendBits(byte, 8));
  appendBits(0, Math.min(4, dataCodewords * 8 - bits.length));
  while (bits.length % 8 !== 0) bits.push(0);
  let pad = 0;
  while (bits.length < dataCodewords * 8) {
    appendBits(pad % 2 === 0 ? 236 : 17, 8);
    pad += 1;
  }
  const data: number[] = [];
  for (let offset = 0; offset < bits.length; offset += 8) {
    let byte = 0;
    for (let bit = 0; bit < 8; bit += 1) byte = (byte << 1) | bits[offset + bit];
    data.push(byte);
  }
  const exponent = new Array<number>(512);
  const logarithm = new Array<number>(256).fill(0);
  let valueAt = 1;
  for (let index = 0; index < 255; index += 1) {
    exponent[index] = valueAt;
    logarithm[valueAt] = index;
    valueAt <<= 1;
    if (valueAt & 256) valueAt ^= 285;
  }
  for (let index = 255; index < 512; index += 1) exponent[index] = exponent[index - 255];
  const multiply = (left: number, right: number) => left === 0 || right === 0 ? 0 : exponent[logarithm[left] + logarithm[right]];
  let generator = [1];
  for (let degree = 0; degree < errorCodewords; degree += 1) {
    const next = new Array<number>(generator.length + 1).fill(0);
    generator.forEach((coefficient, index) => {
      next[index] ^= coefficient;
      next[index + 1] ^= multiply(coefficient, exponent[degree]);
    });
    generator = next;
  }
  const remainder = new Array<number>(errorCodewords).fill(0);
  data.forEach((byte) => {
    const factor = byte ^ remainder[0];
    remainder.shift();
    remainder.push(0);
    for (let index = 0; index < errorCodewords; index += 1) remainder[index] ^= multiply(generator[index + 1], factor);
  });
  const codewordBits: number[] = [];
  data.concat(remainder).forEach((byte) => {
    for (let bit = 7; bit >= 0; bit -= 1) codewordBits.push((byte >>> bit) & 1);
  });
  const matrix = Array.from({ length: size }, () => new Array<boolean>(size).fill(false));
  const functions = Array.from({ length: size }, () => new Array<boolean>(size).fill(false));
  const setFunction = (x: number, y: number, dark: boolean) => {
    if (x >= 0 && y >= 0 && x < size && y < size) {
      matrix[y][x] = dark;
      functions[y][x] = true;
    }
  };
  const drawFinder = (left: number, top: number) => {
    for (let dy = -1; dy <= 7; dy += 1) {
      for (let dx = -1; dx <= 7; dx += 1) {
        const inside = dx >= 0 && dx <= 6 && dy >= 0 && dy <= 6;
        setFunction(left + dx, top + dy, inside && (dx === 0 || dx === 6 || dy === 0 || dy === 6 || (dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4)));
      }
    }
  };
  drawFinder(0, 0);
  drawFinder(size - 7, 0);
  drawFinder(0, size - 7);
  for (let index = 8; index < size - 8; index += 1) {
    if (!functions[6][index]) setFunction(index, 6, index % 2 === 0);
    if (!functions[index][6]) setFunction(6, index, index % 2 === 0);
  }
  for (let dy = -2; dy <= 2; dy += 1) {
    for (let dx = -2; dx <= 2; dx += 1) setFunction(30 + dx, 30 + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
  }
  for (let index = 0; index < 15; index += 1) {
    const verticalY = index < 6 ? index : (index < 8 ? index + 1 : size - 15 + index);
    const horizontalX = index < 8 ? size - index - 1 : (index === 8 ? 7 : 14 - index);
    setFunction(8, verticalY, false);
    setFunction(horizontalX, 8, false);
  }
  setFunction(8, size - 8, true);
  let bitIndex = 0;
  let upward = true;
  for (let right = size - 1; right >= 1; right -= 2) {
    if (right === 6) right -= 1;
    for (let step = 0; step < size; step += 1) {
      const y = upward ? size - 1 - step : step;
      for (let offset = 0; offset < 2; offset += 1) {
        const x = right - offset;
        if (functions[y][x]) continue;
        const raw = bitIndex < codewordBits.length ? codewordBits[bitIndex] === 1 : false;
        matrix[y][x] = raw !== ((x + y) % 2 === 0);
        bitIndex += 1;
      }
    }
    upward = !upward;
  }
  const formatData = 8;
  let remainderBits = formatData << 10;
  for (let bit = 14; bit >= 10; bit -= 1) if ((remainderBits >>> bit) & 1) remainderBits ^= 0x537 << (bit - 10);
  const formatBits = ((formatData << 10) | remainderBits) ^ 0x5412;
  for (let index = 0; index < 15; index += 1) {
    const dark = ((formatBits >>> index) & 1) === 1;
    const verticalY = index < 6 ? index : (index < 8 ? index + 1 : size - 15 + index);
    const horizontalX = index < 8 ? size - index - 1 : (index === 8 ? 7 : 14 - index);
    matrix[verticalY][8] = dark;
    matrix[8][horizontalX] = dark;
  }
  matrix[size - 8][8] = true;
  return matrix;
}

export function render(container: Element, value: string, doc: Document = document): void {
  const matrix = createQRCodeMatrix(value);
  const quietZone = 4;
  const dimension = matrix.length + quietZone * 2;
  const svg = doc.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", `0 0 ${dimension} ${dimension}`);
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", "接收地址二维码");
  svg.setAttribute("shape-rendering", "crispEdges");
  const background = doc.createElementNS("http://www.w3.org/2000/svg", "rect");
  background.setAttribute("width", String(dimension));
  background.setAttribute("height", String(dimension));
  background.setAttribute("fill", "#ffffff");
  svg.appendChild(background);
  const path = doc.createElementNS("http://www.w3.org/2000/svg", "path");
  let commands = "";
  matrix.forEach((row, y) => row.forEach((dark, x) => {
    if (dark) commands += `M${x + quietZone} ${y + quietZone}h1v1h-1z`;
  }));
  path.setAttribute("d", commands);
  path.setAttribute("fill", "#020617");
  svg.appendChild(path);
  container.replaceChildren(svg);
}
