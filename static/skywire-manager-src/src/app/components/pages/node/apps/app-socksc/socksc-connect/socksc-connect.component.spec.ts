import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SockscConnectComponent } from './socksc-connect.component';

describe('SockscConnectComponent', () => {
  let component: SockscConnectComponent;
  let fixture: ComponentFixture<SockscConnectComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SockscConnectComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SockscConnectComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
