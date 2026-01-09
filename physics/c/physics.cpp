#include "physics.h"
#include "util.h"

#include <cassert>
#include <bit>
#include <thread>
#include <span>

#include <Jolt/Jolt.h>

#include <Jolt/Core/Factory.h>
#include <Jolt/Core/JobSystemThreadPool.h>
#include <Jolt/Core/TempAllocator.h>
#include <Jolt/Physics/Body/BodyActivationListener.h>
#include <Jolt/Physics/Body/BodyCreationSettings.h>
#include <Jolt/Physics/Body/BodyLock.h>
#include <Jolt/Physics/Body/BodyLockMulti.h>
#include <Jolt/Physics/Collision/CastResult.h>
#include <Jolt/Physics/Collision/CollideShape.h>
#include <Jolt/Physics/Collision/CollisionGroup.h>
#include <Jolt/Physics/Collision/RayCast.h>
#include <Jolt/Physics/Collision/ShapeCast.h>
#include <Jolt/Physics/Collision/ShapeFilter.h>
#include <Jolt/Physics/PhysicsSettings.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/RegisterTypes.h>

// note: this is only used in the cpp part and so can live inside physics
// namespace
namespace worldspawn {

namespace physics {

// TODO: we'll want to change or make our own JobSystem as the default one uses
// a single queue for all workers
JPH::JobSystemThreadPool jobSystem;

std::once_flag inited;

void init() {
	std::call_once(inited, []() {
		JPH::RegisterDefaultAllocator();

		// Has to be done after JPH::RegisterDefaultAllocator
		jobSystem.Init(JPH::cMaxPhysicsJobs, JPH::cMaxPhysicsBarriers, -1);

		JPH::Factory::sInstance = new JPH::Factory();
		JPH::RegisterTypes();
	});
}

}

}

using namespace worldspawn::physics;

class BroadPhaseLayerInterface final : public JPH::BroadPhaseLayerInterface
{
public:
	BroadPhaseLayerInterface(int broadPhaseLayerCount, int objectLayerCount, std::span<const JPH::BroadPhaseLayer::Type> objectLayerToBroadPhaseLayer) :
		broadPhaseLayerCount(broadPhaseLayerCount),
		objectLayerToBroadPhaseLayer(objectLayerToBroadPhaseLayer.begin(), objectLayerToBroadPhaseLayer.end())
	{
	}

	virtual uint GetNumBroadPhaseLayers() const override
	{
		return broadPhaseLayerCount;
	}

	virtual JPH::BroadPhaseLayer GetBroadPhaseLayer(JPH::ObjectLayer inLayer) const override
	{
		return objectLayerToBroadPhaseLayer[inLayer];
	}

#if defined(JPH_EXTERNAL_PROFILE) || defined(JPH_PROFILE_ENABLED)
	virtual const char *GetBroadPhaseLayerName(BroadPhaseLayer inLayer) const override
	{
		return broadPhaseLayerNames[static_cast<BroadPhaseLayer::Type>(inLayer)].c_str();
	}
#endif

private:
	int broadPhaseLayerCount;
	std::vector<JPH::BroadPhaseLayer> objectLayerToBroadPhaseLayer;
	std::vector<std::string> broadPhaseLayerNames;
};

namespace
{

template<typename T>
bool lowerTriangleIndex(const T &m, size_t i, size_t j) {
	auto row = std::max(i, j);
	auto col = std::min(i, j);
	return m[row * (row + 1) / 2 + col];
}

}

class ObjectVsBroadPhaseLayerFilter : public JPH::ObjectVsBroadPhaseLayerFilter
{
public:
	ObjectVsBroadPhaseLayerFilter(int broadPhaseLayerCount, int objectLayerCount, std::span<const JPH::BroadPhaseLayer::Type> objectLayerToBroadPhaseLayer, std::span<const bool> shouldObjectLayersCollide) :
		broadPhaseLayerCount(broadPhaseLayerCount),
		objectLayerCount(objectLayerCount)
	{
		shouldCollide.resize(objectLayerCount * broadPhaseLayerCount);
		for (int i = 0; i < objectLayerCount; i++) {
			for (int j = 0; j < objectLayerCount; j++) {
				if (lowerTriangleIndex(shouldObjectLayersCollide, i, j))
					shouldCollide[i * broadPhaseLayerCount + objectLayerToBroadPhaseLayer[j]] = true;
			}
		}
	}

	virtual bool ShouldCollide(JPH::ObjectLayer inLayer1, JPH::BroadPhaseLayer inLayer2) const override
	{
		JPH_ASSERT(inLayer1 < objectLayerCount);
		JPH_ASSERT((JPH::BroadPhaseLayer::Type)inLayer2 < broadPhaseLayerCount);
		return shouldCollide[inLayer1 * broadPhaseLayerCount + (JPH::BroadPhaseLayer::Type)inLayer2];
	}

private:
	int broadPhaseLayerCount;
	int objectLayerCount;
	std::vector<bool> shouldCollide;
};

class ObjectLayerPairFilter : public JPH::ObjectLayerPairFilter
{
public:
	ObjectLayerPairFilter(int objectLayerCount, std::span<const bool> shouldCollide) :
		objectLayerCount(objectLayerCount),
		shouldCollide(shouldCollide.begin(), shouldCollide.end())
	{
	}

	virtual bool ShouldCollide(JPH::ObjectLayer inObject1, JPH::ObjectLayer inObject2) const override
	{
		JPH_ASSERT(inObject1 < objectLayerCount);
		JPH_ASSERT(inObject2 < objectLayerCount);
		return lowerTriangleIndex(shouldCollide, inObject1, inObject2);
	}

private:
	int objectLayerCount;
	std::vector<bool> shouldCollide;
};

class ContactListener : public JPH::ContactListener {
public:
	virtual JPH::ValidateResult OnContactValidate(const JPH::Body &inBody1, const JPH::Body &inBody2, JPH::RVec3Arg inBaseOffset, const JPH::CollideShapeResult &inCollisionResult) override
	{
		// TODO: is there any time earlier when we can do this?
		/*
		auto ignores1 = reinterpret_cast<std::vector<JPH::BodyID>*>(inBody1.GetUserData());
		if (ignores1 != nullptr) {
			for (auto id : *ignores1) {
				if (id == inBody2.GetID())
					return JPH::ValidateResult::RejectAllContactsForThisBodyPair;
			}
		}
		auto ignores2 = reinterpret_cast<std::vector<JPH::BodyID>*>(inBody2.GetUserData());
		if (ignores2 != nullptr) {
			for (auto id : *ignores2) {
				if (id == inBody1.GetID())
					return JPH::ValidateResult::RejectAllContactsForThisBodyPair;
			}
		}
		*/
		return JPH::ValidateResult::AcceptAllContactsForThisBodyPair;
	}

	virtual void OnContactAdded(const JPH::Body &inBody1, const JPH::Body &inBody2, const JPH::ContactManifold &inManifold, JPH::ContactSettings &ioSettings) override;

	virtual void OnContactRemoved(const JPH::SubShapeIDPair &inSubShapePair) override;

	// Do we need this thing here? I guess for mapping IDs to bodies in OnContactRemoved?
	Physics *system;

	std::mutex mu; // protects contactEvents
	std::vector<ContactEvent> contactEvents;
};

// PhysicsSystem
struct Physics {
	Physics(int broadPhaseLayerCount, int objectLayerCount, std::span<const JPH::BroadPhaseLayer::Type> objectLayerToBroadPhaseLayer, std::span<const bool> shouldObjectLayersCollide);
	Physics(const Physics&) = delete;
	Physics(const Physics&&) = delete;

	// We should try making the methods be ecs stages along with various
	// queries

	JPH::TempAllocatorMalloc tempAllocator;

	// TODO: we'll probably want to share this between different instances of
	// PhysicsSystem
	// JPH::JobSystemThreadPool jobSystem;

	JPH::PhysicsSystem physicsSystem;

	BroadPhaseLayerInterface broadPhaseLayerInterface;
	ObjectVsBroadPhaseLayerFilter objectVsBroadphaseLayerFilter;
	ObjectLayerPairFilter objectLayerPairFilter;
	ContactListener contactListener;

	JPH::BodyIDVector activeBodies;
};

Physics::Physics(int broadPhaseLayerCount, int objectLayerCount, std::span<const JPH::BroadPhaseLayer::Type> objectLayerToBroadPhaseLayer, std::span<const bool> shouldObjectLayersCollide) :
	// jobSystem(JPH::cMaxPhysicsJobs, JPH::cMaxPhysicsBarriers, -1),
	broadPhaseLayerInterface(broadPhaseLayerCount, objectLayerCount, objectLayerToBroadPhaseLayer),
	objectVsBroadphaseLayerFilter(broadPhaseLayerCount, objectLayerCount, objectLayerToBroadPhaseLayer, shouldObjectLayersCollide),
	objectLayerPairFilter(objectLayerCount, shouldObjectLayersCollide)
{
	contactListener.system = this;

	physicsSystem.Init(
		16384, // max bodies
		0,     // body mutexes
		16384, // max body pairs
		16384, // max constraints
		broadPhaseLayerInterface,
		objectVsBroadphaseLayerFilter,
		objectLayerPairFilter
	);
	physicsSystem.SetContactListener(&contactListener);
}

void ContactListener::OnContactAdded(const JPH::Body &inBody1, const JPH::Body &inBody2, const JPH::ContactManifold &inManifold, JPH::ContactSettings &ioSettings)
{
	auto ce = ContactEvent{
		.type = ContactAdded,
		.body1 = {
			.bodyID = inBody1.GetID().GetIndexAndSequenceNumber(),
			.subShapeID = inManifold.mSubShapeID1.GetValue(),
			.active = inBody1.IsActive(),
		},
		.body2 = {
			.bodyID = inBody2.GetID().GetIndexAndSequenceNumber(),
			.subShapeID = inManifold.mSubShapeID2.GetValue(),
			.active = inBody2.IsActive(),
		},
		.normal = JPHVec3ToVec3(inManifold.mWorldSpaceNormal),
	};

	std::lock_guard<std::mutex> lock(mu);

	contactEvents.push_back(ce);
}

void ContactListener::OnContactRemoved(const JPH::SubShapeIDPair &inSubShapePair) {
	auto &bodyLockInterface = system->physicsSystem.GetBodyLockInterfaceNoLock();

	auto inBody1 = bodyLockInterface.TryGetBody(inSubShapePair.GetBody1ID());
	auto inBody2 = bodyLockInterface.TryGetBody(inSubShapePair.GetBody2ID());

	auto ce = ContactEvent{
		.type = ContactRemoved,
		.body1 = {
			.bodyID = inSubShapePair.GetBody1ID().GetIndexAndSequenceNumber(),
			.subShapeID = inSubShapePair.GetSubShapeID1().GetValue(),
			.active = inBody1 ? inBody1->IsActive() : false, // TODO: we should communicate when a contact is being removed because a body was removed
		},
		.body2 = {
			.bodyID = inSubShapePair.GetBody2ID().GetIndexAndSequenceNumber(),
			.subShapeID = inSubShapePair.GetSubShapeID2().GetValue(),
			.active = inBody2 ? inBody2->IsActive() : false,
		},
	};

	std::lock_guard<std::mutex> lock(mu);

	contactEvents.push_back(ce);
}

Physics* newPhysics(
	int broadPhaseLayerCount,
	int objectLayerCount,
	const uint8_t *objectLayerToBroadPhaseLayer,
	const bool *shouldObjectLayersCollide) {
	worldspawn::physics::init();

	return new Physics(
		broadPhaseLayerCount,
		objectLayerCount,
		std::span<const JPH::BroadPhaseLayer::Type>(objectLayerToBroadPhaseLayer, objectLayerToBroadPhaseLayer + objectLayerCount),
		std::span<const bool>(shouldObjectLayersCollide, shouldObjectLayersCollide + objectLayerCount * (objectLayerCount + 1) / 2));
}

namespace
{

JPH::BodyCreationSettings motionPropertiesToJPHBodyCreationSettings(MotionProperties motionProperties)
{
	JPH::BodyCreationSettings bodyCreationSettings;

	bodyCreationSettings.SetShape(reinterpret_cast<const JPH::Shape*>(motionProperties.shape));

	bodyCreationSettings.mPosition = dvec3ToJPHDVec3(motionProperties.motionState.position);
	bodyCreationSettings.mRotation = rotation3ToJPHQuat(motionProperties.motionState.rotation);
	bodyCreationSettings.mLinearVelocity = vec3ToJPHVec3(motionProperties.motionState.velocity);
	bodyCreationSettings.mAngularVelocity = vec3ToJPHVec3(motionProperties.motionState.angularVelocity);

	bodyCreationSettings.mObjectLayer = motionProperties.objectLayer;

	bodyCreationSettings.mMotionType = static_cast<JPH::EMotionType>(motionProperties.motionType);
	bodyCreationSettings.mAllowDynamicOrKinematic = true;
	bodyCreationSettings.mMotionQuality = JPH::EMotionQuality::LinearCast; // TODO: plumb motion quality explicitly, we want lower motion quality for cosmetic things like ragdolls
	bodyCreationSettings.mEnhancedInternalEdgeRemoval = true;
	// TODO: set mass, friction, restitution, etc, from the material
	bodyCreationSettings.mFriction = 0.5f;
	// bodyCreationSettings.mRestitution = 0.8f;
	bodyCreationSettings.mGravityFactor = motionProperties.gravityFactor;

	bodyCreationSettings.mOverrideMassProperties = JPH::EOverrideMassProperties::MassAndInertiaProvided;
	bodyCreationSettings.mMassPropertiesOverride = JPH::MassProperties{
		.mMass = motionProperties.mass,
		.mInertia = mat4x4ToJPHMat44(motionProperties.inertia),
	};

	return bodyCreationSettings;
}

JPH::RMat44 GetCenterOfMassTransform(JPH::RVec3Arg inPosition, JPH::QuatArg inRotation, const JPH::Shape *inShape)
{
	return JPH::RMat44::sRotationTranslation(inRotation, inPosition).PreTranslated(inShape->GetCenterOfMass()); // .PostTranslated(mCharacterPadding * mUp);
}

}

// TODO: to implement per-subshape friction and restitution we just need to set
// custom friction and restitution combine functions which will pull stuff out
// of subshapes. We'll also want to disable "manifold combining" or whatever it
// was called for select shapes.

/*
class Material : public JPH::PhysicsMaterial {
public:

private:
	float friction;
	float restitution;
	float density;
};
*/

class MyBodyFilter : public JPH::BodyFilter
{
public:
	MyBodyFilter(JPH::BodyID ignore) : ignore(ignore) {}

	virtual bool ShouldCollide(const JPH::BodyID &inBodyID) const override
	{
		return inBodyID != ignore;
	}

private:
	// TODO: specify bodies we pretend to be so as to apply those body's filter
	JPH::BodyID ignore;
};

void physicsQueryRayClosestHit(Physics *system, const dvec3 *pos, const vec3 *dir) {
	JPH::RayCastResult result;
}

size_t physicsQueryShape(Physics *system, const Shape *shape_, const dvec3 *pos, const Rot3 *rot, const vec3 *scale, const vec3 *movementDirection, float maxSeparationDistance, QueryFilter filter, QueryHit *outHits, size_t maxHits) {
	class Collector : public JPH::CollideShapeCollector {
	public:
		// TODO: option for contact reduction

		virtual void AddHit(const JPH::CollideShapeResult &result) override {
			if (i >= out.size()) {
				ForceEarlyOut();
				return;
			}

			out[i++] = QueryHit{
				.point  = JPHDVec3ToDVec3(baseOffset + result.mContactPointOn2),
				.normal = JPHVec3ToVec3(result.mPenetrationAxis.NormalizedOr(JPH::Vec3::sZero())),
				.depth  = result.mPenetrationDepth,
			};
		}

		JPH::RVec3 baseOffset;
		std::span<QueryHit> out;
		size_t i = 0;
	};

	auto shape = reinterpret_cast<const JPH::Shape*>(shape_);

	JPH::CollideShapeSettings settings;
	settings.mActiveEdgeMovementDirection = vec3ToJPHVec3(*movementDirection);
	settings.mMaxSeparationDistance = maxSeparationDistance;

	Collector collector;
	collector.baseOffset = dvec3ToJPHDVec3(*pos);
	collector.out = std::span<QueryHit>(outHits, outHits + maxHits);

	system->physicsSystem.GetNarrowPhaseQuery().CollideShape(
		shape,
		vec3ToJPHVec3(*scale),
		GetCenterOfMassTransform(dvec3ToJPHDVec3(*pos), rotation3ToJPHQuat(*rot), shape),
		settings,
		dvec3ToJPHDVec3(*pos),
		collector,
		{},
		{},
		MyBodyFilter(JPH::BodyID(filter.ignore)),
		{});

	return collector.i;
}

CastQueryResult physicsQuerySweptShapeClosestHit(Physics *system, const Shape *shape_, const dvec3 *pos, const Rot3 *rot, const vec3 *scale, const vec3 *displacement, QueryFilter filter) {
	class Collector : public JPH::CastShapeCollector {
	public:
		virtual void AddHit(const JPH::ShapeCastResult &result) override {
			hit = CastQueryResult{
				.fraction = result.mFraction,
				.base = {
					.point  = JPHDVec3ToDVec3(baseOffset + result.mContactPointOn2),
					.normal = JPHVec3ToVec3(result.mPenetrationAxis.NormalizedOr(JPH::Vec3::sZero())),
					.depth  = result.mPenetrationDepth,
				},
			};

			UpdateEarlyOutFraction(result.mFraction);
		}

		JPH::RVec3 baseOffset;
		CastQueryResult hit;
	};

	auto shape = reinterpret_cast<const JPH::Shape*>(shape_);

	JPH::ShapeCastSettings settings;

	Collector collector;
	collector.baseOffset = dvec3ToJPHDVec3(*pos);
	collector.hit = CastQueryResult{.fraction = 1.0f + FLT_EPSILON};

	system->physicsSystem.GetNarrowPhaseQuery().CastShape(
		JPH::RShapeCast(
			shape,
			vec3ToJPHVec3(*scale),
			GetCenterOfMassTransform(dvec3ToJPHDVec3(*pos), rotation3ToJPHQuat(*rot), shape),
			vec3ToJPHVec3(*displacement)),
		settings,
		dvec3ToJPHDVec3(*pos),
		collector,
		{},
		{},
		MyBodyFilter(JPH::BodyID(filter.ignore)),
		{});

	return collector.hit;
}

void physicsSetGravity(Physics *system, const vec3 *gravity) {
	system->physicsSystem.SetGravity(vec3ToJPHVec3(*gravity));
}

// TODO: the user should provide the container to write results (active bodies
// and contact events) into
void physicsUpdate(Physics *system, float dt) {
	system->contactListener.contactEvents.clear();
	system->physicsSystem.Update(dt, 1, &system->tempAllocator, &jobSystem);
	system->physicsSystem.GetActiveBodies(JPH::EBodyType::RigidBody, system->activeBodies);
}

// TODO: convert this api to multi-body

void physicsAddBody(Physics *system, BodyID bodyID_, MotionProperties motionProperties) {
	auto &bodyInterface = system->physicsSystem.GetBodyInterfaceNoLock();

	auto body = bodyInterface.CreateBodyWithID(JPH::BodyID(bodyID_), motionPropertiesToJPHBodyCreationSettings(motionProperties));
	assert(body != nullptr);

	if (motionProperties.ignoreBodyCount > 0) {
		// TODO: we should reinterpret uint64_t as a type with destructor so
		// that replacing it doesn't induce headaches
		// TODO: apply this when updating the body as well
		//body->SetUserData(reinterpret_cast<uint64_t>(new std::vector<JPH::BodyID>(motionProperties.ignoreBodies, motionProperties.ignoreBodies + motionProperties.ignoreBodyCount)));
	}

	// TODO: append this to a list of bodies to add instead of adding
	// immediately when this function becomes multi-body
	bodyInterface.AddBody(body->GetID(), JPH::EActivation::Activate);
}

// Our app should be the one tracking whether to add bodies or delete or what.
void physicsUpdateBody(Physics *system, BodyID bodyID_, MotionProperties motionProperties) {
	auto &bodyLockInterface = system->physicsSystem.GetBodyLockInterface();
	auto &bodyInterface = system->physicsSystem.GetBodyInterfaceNoLock();

	// TODO: use multi lock when this function becomes multi-body
	JPH::BodyLockWrite lock(bodyLockInterface, JPH::BodyID(bodyID_));
	assert(lock.Succeeded());
	auto &body = lock.GetBody();

	// TODO: we should use the body methods directly for things that
	// are possible (velocity, force, ...), but we'll have to
	// activate it manually

	// TODO:

	// TODO: record activation reason also
	bool activate = false;

	auto bodyCreationSettings = motionPropertiesToJPHBodyCreationSettings(motionProperties);

	if (body.GetShape() != bodyCreationSettings.GetShape()) {
		bodyInterface.SetShape(
			body.GetID(),
			bodyCreationSettings.GetShape(),
			false, // we update mass properties ourselves
			JPH::EActivation::DontActivate);

		activate = true;
	}

	bodyInterface.SetPositionAndRotationWhenChanged(
		body.GetID(),
		bodyCreationSettings.mPosition,
		bodyCreationSettings.mRotation,
		JPH::EActivation::Activate);
	if (!body.IsStatic()) {
		body.SetLinearVelocityClamped(bodyCreationSettings.mLinearVelocity);
		body.SetAngularVelocityClamped(bodyCreationSettings.mAngularVelocity);
		if (!bodyCreationSettings.mLinearVelocity.IsNearZero() ||
			!bodyCreationSettings.mAngularVelocity.IsNearZero())
			activate = true;
	}

	bodyInterface.SetObjectLayer(body.GetID(), bodyCreationSettings.mObjectLayer);
	body.SetCollisionGroup(bodyCreationSettings.mCollisionGroup);

	if (bodyInterface.GetMotionType(body.GetID()) != bodyCreationSettings.mMotionType) {
		bodyInterface.SetMotionType(body.GetID(), bodyCreationSettings.mMotionType, JPH::EActivation::DontActivate);
		activate = true;
	}

	if (!body.IsStatic()) {
		auto motionProperties = body.GetMotionProperties();

		// TODO: set friction, restitution from the material

		// TODO: activate the body if anything here actually changes

		motionProperties->SetGravityFactor(bodyCreationSettings.mGravityFactor);
		motionProperties->SetMassProperties(JPH::EAllowedDOFs::All, bodyCreationSettings.mMassPropertiesOverride);
	}

	if (activate) {
		// TODO: add this body to the list of bodies to activate instead of activating immediately
		bodyInterface.ActivateBody(body.GetID());
	}
}

// TODO: similarly, convert this to a multi body
void physicsRemoveBody(Physics *system, BodyID bodyID) {
	auto &bodyLockInterface = system->physicsSystem.GetBodyLockInterface();
	auto &bodyInterface = system->physicsSystem.GetBodyInterfaceNoLock();

	JPH::BodyLockWrite lock(bodyLockInterface, JPH::BodyID(bodyID));
	assert(lock.Succeeded());
	auto &body = lock.GetBody();

	// This looks ugly
	//delete reinterpret_cast<std::vector<JPH::BodyID>*>(body.GetUserData());

	bodyInterface.RemoveBody(body.GetID());
	bodyInterface.DestroyBody(body.GetID());
}

void physicsWritebackBody(Physics *system, BodyID bodyID, MotionState *out) {
	// TODO: make this multi-body

	auto &bodyLockInterface = system->physicsSystem.GetBodyLockInterface();

	JPH::BodyLockRead lock(bodyLockInterface, JPH::BodyID(bodyID));

	auto &body = lock.GetBody();

	out->position = JPHDVec3ToDVec3(body.GetPosition());

	out->rotation.scalar = body.GetRotation().GetW();
	out->rotation.yz = body.GetRotation().GetX();
	out->rotation.zx = body.GetRotation().GetY();
	out->rotation.xy = body.GetRotation().GetZ();

	out->velocity = JPHVec3ToVec3(body.GetLinearVelocity());
	out->angularVelocity = JPHVec3ToVec3(body.GetAngularVelocity());
}

const BodyID* physicsActiveBodies(Physics *system, size_t *activeBodyCount)
{
	*activeBodyCount = system->activeBodies.size();
	return reinterpret_cast<BodyID*>(system->activeBodies.data());
}

const ContactEvent* physicsContactEvents(Physics *system, size_t *contactEventCount)
{
	*contactEventCount = system->contactListener.contactEvents.size();
	return system->contactListener.contactEvents.data();
}

// TODO: make helpers for working with JPH::Ref

// TODO: we need two functions to create shapes: one would be creating from raw
// shapes, another one would be deserializing Jolt shapes as is, which we'll be
// caching

