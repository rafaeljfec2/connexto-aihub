class ResizeObserverMock {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}

globalThis.ResizeObserver = ResizeObserverMock as typeof ResizeObserver

Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
  const element = this as HTMLElement
  const width = element.offsetWidth || 560
  const height = element.offsetHeight || 120

  return {
    width,
    height,
    top: 0,
    left: 0,
    right: width,
    bottom: height,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  }
}
