import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { StartupConfigComponent } from './startup-config.component';

describe('StartupConfigComponent', () => {
  let component: StartupConfigComponent;
  let fixture: ComponentFixture<StartupConfigComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ StartupConfigComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(StartupConfigComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
