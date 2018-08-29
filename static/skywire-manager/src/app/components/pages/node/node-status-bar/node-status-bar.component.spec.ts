import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NodeStatusBarComponent } from './node-status-bar.component';

describe('StatusBarComponent', () => {
  let component: NodeStatusBarComponent;
  let fixture: ComponentFixture<NodeStatusBarComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NodeStatusBarComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NodeStatusBarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
