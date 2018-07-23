import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NodeTransportsList } from './node-transports-list';

describe('NodeTransportsList', () => {
  let component: NodeTransportsList;
  let fixture: ComponentFixture<NodeTransportsList>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NodeTransportsList ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NodeTransportsList);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
