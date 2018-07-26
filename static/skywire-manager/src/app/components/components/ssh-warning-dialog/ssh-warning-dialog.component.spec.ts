import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SshWarningDialogComponent } from './ssh-warning-dialog.component';

describe('SshWarningDialogComponent', () => {
  let component: SshWarningDialogComponent;
  let fixture: ComponentFixture<SshWarningDialogComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SshWarningDialogComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SshWarningDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
