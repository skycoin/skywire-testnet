import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SshcStartupComponent } from './sshc-startup.component';

describe('SshcStartupComponent', () => {
  let component: SshcStartupComponent;
  let fixture: ComponentFixture<SshcStartupComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SshcStartupComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SshcStartupComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
