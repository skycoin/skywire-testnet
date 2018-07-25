import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SshsStartupComponent } from './sshs-startup.component';

describe('SshsStartupComponent', () => {
  let component: SshsStartupComponent;
  let fixture: ComponentFixture<SshsStartupComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SshsStartupComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SshsStartupComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
