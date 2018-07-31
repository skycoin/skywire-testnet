import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { AppsSettingsComponent } from './apps-settings.component';

describe('AppsSettingsComponent', () => {
  let component: AppsSettingsComponent;
  let fixture: ComponentFixture<AppsSettingsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AppsSettingsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AppsSettingsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
