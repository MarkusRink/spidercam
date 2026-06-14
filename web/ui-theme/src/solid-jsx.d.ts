import type { JSX as SolidJSX } from "solid-js";

declare global {
  namespace JSX {
    type Element = SolidJSX.Element;
    type ArrayElement = SolidJSX.ArrayElement;
    type ElementClass = SolidJSX.ElementClass;
    type ElementAttributesProperty = SolidJSX.ElementAttributesProperty;
    type ElementChildrenAttribute = SolidJSX.ElementChildrenAttribute;
    type IntrinsicElements = SolidJSX.IntrinsicElements;
  }
}

export {};
