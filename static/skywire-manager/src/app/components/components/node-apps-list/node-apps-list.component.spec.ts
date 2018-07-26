import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NodeAppsListComponent } from './node-apps-list.component';

describe('NodeAppsListComponent', () => {
  let component: NodeAppsListComponent;
  let fixture: ComponentFixture<NodeAppsListComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NodeAppsListComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NodeAppsListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
