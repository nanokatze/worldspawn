#include "physics.h"
#include "util.h"

#include <cassert>

#include <Jolt/Jolt.h>

#include <Jolt/Physics/Collision/Shape/BoxShape.h>
#include <Jolt/Physics/Collision/Shape/CapsuleShape.h>
#include <Jolt/Physics/Collision/Shape/CompoundShape.h>
#include <Jolt/Physics/Collision/Shape/ConvexHullShape.h>
#include <Jolt/Physics/Collision/Shape/CylinderShape.h>
#include <Jolt/Physics/Collision/Shape/MeshShape.h>
#include <Jolt/Physics/Collision/Shape/RotatedTranslatedShape.h>
#include <Jolt/Physics/Collision/Shape/SphereShape.h>

using namespace worldspawn::physics;

namespace worldspawn::physics {

void init();

}

namespace
{

Shape*
newShape(const JPH::ShapeSettings &shapeSettings)
{
	worldspawn::physics::init();

	auto result = shapeSettings.Create();

	if (result.HasError()) {
		// TODO: propagate this instead of crashing
		fprintf(stderr, "%s\n", result.GetError().c_str());
		assert(false);
	}

	auto shape = result.Get();

	if (!shape->MustBeStatic()) {
		auto massProps = shape->GetMassProperties();

		if (isnanf(massProps.mMass)) {
			assert(0);
		}
	}

	return static_cast<Shape*>(std::exchange(*shape.InternalGetPointer(), nullptr));
}

}

Shape*
newSphereShape(float radius)
{
	return newShape(JPH::SphereShapeSettings(radius));
}

Shape*
newBoxShape(const vec3 *halfExtent, float convexRadius)
{
	return newShape(JPH::BoxShapeSettings(vec3ToJPHVec3(*halfExtent), convexRadius));
}

Shape*
newCylinderShape(float radius, float halfHeight, float convexRadius)
{
	auto cylinder = newShape(JPH::CylinderShapeSettings(halfHeight, radius, convexRadius));
	return newShape(JPH::RotatedTranslatedShapeSettings(JPH::Vec3::sZero(), JPH::Quat(sqrt(0.5), 0, 0, sqrt(0.5)), reinterpret_cast<JPH::Shape*>(cylinder)));
}

Shape*
newConvexHullShape(const vec3 *points, size_t pointCount, float convexRadius)
{
	JPH::Array<JPH::Vec3> tmp;
	tmp.reserve(pointCount);
	for (size_t i = 0; i < pointCount; i++)
		tmp.push_back(vec3ToJPHVec3(points[i]));

	return newShape(JPH::ConvexHullShapeSettings(tmp, convexRadius));
}

Shape*
newMeshShape(const vec3 *vertices, size_t vertexCount, const void *triangles, size_t triangleCount)
{
	JPH::VertexList tmp;
	tmp.reserve(vertexCount);
	for (size_t i = 0; i < vertexCount; i++)
		tmp.push_back(JPH::Float3(vertices[i].x, vertices[i].y, vertices[i].z));

	JPH::IndexedTriangleList tmp2(
		static_cast<const JPH::IndexedTriangle*>(triangles),
		static_cast<const JPH::IndexedTriangle*>(triangles)+triangleCount);

	return newShape(JPH::MeshShapeSettings(tmp, tmp2));
}

Shape*
newTransformedShape(const vec3 *translation, const Rot3 *rotation, const vec3 *scale, const Shape *shape)
{
	// TODO: scale
	return newShape(JPH::RotatedTranslatedShapeSettings(vec3ToJPHVec3(*translation), rotation3ToJPHQuat(*rotation), reinterpret_cast<const JPH::Shape*>(shape)));
}

float
shapeMass(const Shape* shape_)
{
	auto shape = reinterpret_cast<const JPH::Shape*>(shape_);
	if (shape->MustBeStatic())
		return 1.0f; // TODO: we should just avoid calling Mass on always static shapes
	return shape->GetMassProperties().mMass;
}

mat4
shapeInertia(const Shape *shape_)
{
	auto shape = reinterpret_cast<const JPH::Shape*>(shape_);
	if (shape->MustBeStatic())
		return JPHMat44ToMat4x4(JPH::Mat44::sIdentity()); // TODO: we should just avoid calling this on always static shapes
	return JPHMat44ToMat4x4(shape->GetMassProperties().mInertia);
}

void
shapeDecRef(Shape* cshape)
{
	JPH::ShapeRefC shape;
	*shape.InternalGetPointer() = cshape;
}
