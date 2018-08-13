import {Directive, ViewContainerRef} from '@angular/core';

@Directive({
  selector: '[component-host]'
})
export class ComponentHostDirective
{
  constructor(public viewContainerRef: ViewContainerRef) { }
}
