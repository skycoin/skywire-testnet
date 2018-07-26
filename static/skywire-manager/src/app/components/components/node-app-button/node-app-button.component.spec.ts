import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NodeAppButtonComponent } from './node-app-button.component';

describe('NodeAppButtonComponent', () => {
  let component: NodeAppButtonComponent;
  let fixture: ComponentFixture<NodeAppButtonComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NodeAppButtonComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NodeAppButtonComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
