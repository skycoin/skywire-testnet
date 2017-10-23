import { Injectable, TemplateRef, ComponentRef, ComponentFactory, ComponentFactoryResolver, Injector } from '@angular/core';
import { ModalWindow } from './modal-window.component'

@Injectable()
export class ModalService {
  private _windowFactory: ComponentFactory<ModalWindow>;
  constructor(private _injector: Injector, private _componentFactoryResolver: ComponentFactoryResolver) {
    this._windowFactory = _componentFactoryResolver.resolveComponentFactory(ModalWindow);
  }

  open(content: any) {
    let windowCmpRef: ComponentRef<ModalWindow>
    const containerEl = document.querySelector('body');
    const viewRef = content.createEmbeddedView(content);
    windowCmpRef = this._windowFactory.create(this._injector, [viewRef.rootNodes])
    containerEl.appendChild(windowCmpRef.location.nativeElement);
    return windowCmpRef;
  }
}
