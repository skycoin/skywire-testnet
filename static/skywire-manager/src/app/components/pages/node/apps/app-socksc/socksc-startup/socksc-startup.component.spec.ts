import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SockscStartupComponent } from './socksc-startup.component';

describe('SockscStartupComponent', () => {
  let component: SockscStartupComponent;
  let fixture: ComponentFixture<SockscStartupComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SockscStartupComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SockscStartupComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
