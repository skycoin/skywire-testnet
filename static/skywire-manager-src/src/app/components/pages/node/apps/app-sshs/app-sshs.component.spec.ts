import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { AppSshsComponent } from './app-sshs.component';

describe('AppSshsComponent', () => {
  let component: AppSshsComponent;
  let fixture: ComponentFixture<AppSshsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AppSshsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AppSshsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
