export function shallowCopy<T>(obj: T): T {
  const copy = Object.create(Object.getPrototypeOf(obj));
  return Object.assign(copy, obj);
}
