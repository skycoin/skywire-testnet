import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { SkycoinLogoComponent } from './skycoin-logo.component';

describe('SkycoinLogoComponent', () => {
  let component: SkycoinLogoComponent;
  let fixture: ComponentFixture<SkycoinLogoComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ SkycoinLogoComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(SkycoinLogoComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
