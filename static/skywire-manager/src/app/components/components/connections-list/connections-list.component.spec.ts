import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ConnectionsListComponent } from './connections-list.component';

describe('ConnectionsListComponent', () => {
  let component: ConnectionsListComponent;
  let fixture: ComponentFixture<ConnectionsListComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ConnectionsListComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ConnectionsListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
