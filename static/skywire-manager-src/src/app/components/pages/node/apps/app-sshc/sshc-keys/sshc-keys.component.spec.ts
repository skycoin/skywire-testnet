import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SshcKeysComponent } from './sshc-keys.component';

describe('SshcKeysComponent', () => {
  let component: SshcKeysComponent;
  let fixture: ComponentFixture<SshcKeysComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SshcKeysComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SshcKeysComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
