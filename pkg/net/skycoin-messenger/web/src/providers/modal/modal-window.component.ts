import { Component, OnInit } from '@angular/core';

@Component({
  // tslint:disable-next-line:component-selector
  selector: 'modal-window',
  template: `
   <div role="document">
        <div class="modal-content"><ng-content></ng-content></div>
    </div>`
})

// tslint:disable-next-line:component-class-suffix
export class ModalWindow implements OnInit {
  constructor() { }

  ngOnInit() { }
}
