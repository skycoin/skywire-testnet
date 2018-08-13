import {Component, ComponentFactoryResolver, Input, OnInit, Type, ViewChild, ViewContainerRef} from '@angular/core';
import {ComponentHostDirective} from "../../../directives/component-host.directive";

@Component({
  selector: 'app-host',
  templateUrl: './host.component.html',
  styleUrls: ['./host.component.css']
})
export class HostComponent implements OnInit
{
  @Input() componentClass: Type<any>;
  @Input() data: any;
  @ViewChild(ComponentHostDirective) host: ComponentHostDirective;

  constructor(
    private componentFactoryResolver: ComponentFactoryResolver
  ) { }

  ngOnInit()
  {
    let componentFactory = this.componentFactoryResolver.resolveComponentFactory(this.componentClass);

    let viewContainerRef = this.host.viewContainerRef;
    viewContainerRef.clear();
    let comp = viewContainerRef.createComponent(componentFactory);

    comp.instance.data = this.data;
  }
}
