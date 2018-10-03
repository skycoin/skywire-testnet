import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SshsWhitelistComponent } from './sshs-whitelist.component';

describe('SshsWhitelistComponent', () => {
  let component: SshsWhitelistComponent;
  let fixture: ComponentFixture<SshsWhitelistComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SshsWhitelistComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SshsWhitelistComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
