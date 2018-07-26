import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { AppSockscComponent } from './app-socksc.component';

describe('AppSockscComponent', () => {
  let component: AppSockscComponent;
  let fixture: ComponentFixture<AppSockscComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AppSockscComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AppSockscComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
