import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { AppSshcComponent } from './app-sshc.component';

describe('AppSshcComponent', () => {
  let component: AppSshcComponent;
  let fixture: ComponentFixture<AppSshcComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AppSshcComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AppSshcComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
