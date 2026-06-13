import type { JSX as SolidJSX } from "solid-js";

declare global {
  namespace JSX {
    type Element = SolidJSX.Element;
    interface ArrayElement extends SolidJSX.ArrayElement {}
    interface ElementClass extends SolidJSX.ElementClass {}
    interface ElementAttributesProperty
      extends SolidJSX.ElementAttributesProperty {}
    interface ElementChildrenAttribute
      extends SolidJSX.ElementChildrenAttribute {}
    interface IntrinsicElements extends SolidJSX.IntrinsicElements {}
  }
}

export {};
